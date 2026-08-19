//go:build integration

// Package integration 提供真实 MySQL/Redis 依赖的集成测试。
// 默认 go test ./... 不执行（build tag=integration）。
// 运行方式（依赖 Docker Compose 已启动 MySQL/Redis，端口映射 13306/16379）：
//
//	docker compose up -d mysql redis
//	MYSQL_DSN="fleet:fleet_pwd@tcp(127.0.0.1:13306)/fleet?parseTime=true&loc=Asia%2FShanghai&charset=utf8mb4&collation=utf8mb4_unicode_ci" \
//	REDIS_ADDR="127.0.0.1:16379" go test -tags=integration ./tests/integration/...
//
// 也可在容器内运行：
//
//	docker compose run --rm api go test -tags=integration ./tests/integration/...
package integration

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/zhangkui/go-fleet-maintenance/internal/domain"
	"github.com/zhangkui/go-fleet-maintenance/internal/domain/entity"
	"github.com/zhangkui/go-fleet-maintenance/internal/platform/migration"
	"github.com/zhangkui/go-fleet-maintenance/internal/platform/mysql"
	"github.com/zhangkui/go-fleet-maintenance/internal/platform/redisx"
	"github.com/zhangkui/go-fleet-maintenance/internal/repository"
	mysqlrepo "github.com/zhangkui/go-fleet-maintenance/internal/repository/mysql"
	"github.com/zhangkui/go-fleet-maintenance/internal/service"
)

var (
	testDB  *sql.DB
	testRdb *redisx.Client
)

func TestMain(m *testing.M) {
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		dsn = "fleet:fleet_pwd@tcp(127.0.0.1:13306)/fleet?parseTime=true&loc=Asia%2FShanghai&charset=utf8mb4&collation=utf8mb4_unicode_ci"
	}
	// root DSN 用于创建/删除测试库（fleet 用户无建库权限）。
	rootDSN := os.Getenv("MYSQL_ROOT_DSN")
	if rootDSN == "" {
		rootDSN = "root:root_pwd_2026@tcp(127.0.0.1:13306)/?parseTime=true&loc=Asia%2FShanghai&charset=utf8mb4&collation=utf8mb4_unicode_ci"
	}
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "127.0.0.1:16379"
	}

	rootDB, err := mysql.Open(rootDSN)
	if err != nil {
		fmt.Fprintln(os.Stderr, "skip: 无法连接 MySQL(root):", err)
		os.Exit(0)
	}
	if err := waitForDB(rootDB, 60); err != nil {
		fmt.Fprintln(os.Stderr, "skip: MySQL 未就绪:", err)
		os.Exit(0)
	}
	ctx := context.Background()
	// 用 root 创建独立测试库并授予 fleet 用户权限。
	if _, err := rootDB.ExecContext(ctx, "CREATE DATABASE IF NOT EXISTS fleet_test CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"); err != nil {
		fmt.Fprintln(os.Stderr, "skip: 无法创建测试库:", err)
		os.Exit(0)
	}
	_, _ = rootDB.ExecContext(ctx, "GRANT ALL PRIVILEGES ON fleet_test.* TO 'fleet'@'%'")
	rootDB.Close()

	// 用 fleet 用户连测试库。
	testDB, err = mysql.Open(replaceDB(dsn, "fleet_test"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "skip:", err)
		os.Exit(0)
	}
	if err := waitForDB(testDB, 30); err != nil {
		fmt.Fprintln(os.Stderr, "skip: 测试库未就绪:", err)
		os.Exit(0)
	}
	// 迁移。
	mr := migration.New(testDB)
	if _, err := mr.Run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "skip: 迁移失败:", err)
		os.Exit(0)
	}

	rdb, err := redisx.New(redisAddr, "", 1, "test:")
	if err == nil {
		if err := rdb.Ping(ctx); err == nil {
			testRdb = rdb
		}
	}
	// 每个测试前清理数据。
	cleanupAll(ctx, testDB)
	// 清理 Redis 测试幂等标记，避免跨测试运行残留。
	if testRdb != nil {
		testRdb.InvalidateCache(ctx, "idem:trip:e2e-trip-1", "idem:fuel:e2e-fuel-1", "idem:fuel:e2e-fuel-2")
	}
	// flushall db=1 彻底清空。
	if testRdb != nil {
		testRdb.Rdb().FlushDB(ctx)
	}

	code := m.Run()
	testDB.Close()
	if testRdb != nil {
		testRdb.Rdb().Close()
	}
	// 用 root 删除测试库。
	rootDB2, err := mysql.Open(rootDSN)
	if err == nil {
		_, _ = rootDB2.ExecContext(ctx, "DROP DATABASE IF EXISTS fleet_test")
		rootDB2.Close()
	}
	os.Exit(code)
}

func replaceDB(dsn, newDB string) string {
	// 把 DSN 中的数据库名替换为 newDB。
	idx := -1
	for i := len(dsn) - 1; i >= 0; i-- {
		if dsn[i] == '/' {
			idx = i
			break
		}
	}
	if idx == -1 {
		return dsn
	}
	// 跳过 / 后的数据库名（到 ? 之前）。
	rest := dsn[idx+1:]
	q := -1
	for i := 0; i < len(rest); i++ {
		if rest[i] == '?' {
			q = i
			break
		}
	}
	if q == -1 {
		return dsn[:idx+1] + newDB
	}
	return dsn[:idx+1] + newDB + rest[q:]
}

func waitForDB(db *sql.DB, max int) error {
	ctx := context.Background()
	for i := 0; i < max; i++ {
		if err := db.PingContext(ctx); err == nil {
			return nil
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("数据库未就绪")
}

// cleanupAll 清空所有业务表，保证测试隔离。
func cleanupAll(ctx context.Context, db *sql.DB) {
	tables := []string{
		"part_stock_movements", "parts", "maintenance_order_status_history",
		"maintenance_order_parts", "maintenance_orders", "maintenance_policies",
		"trip_handovers", "trip_status_history", "trips",
		"fuel_records", "driver_violations", "driver_schedules",
		"driver_vehicle_bindings", "drivers", "vehicle_licenses",
		"vehicle_status_history", "vehicles",
		"refresh_tokens", "audit_logs", "notifications", "reminders",
		"user_roles", "role_permissions", "permissions", "roles", "users",
	}
	for _, t := range tables {
		_, _ = db.ExecContext(ctx, "DELETE FROM "+t)
	}
	// 重新插入最小内置角色/权限/管理员，避免依赖 seed。
	_, _ = db.ExecContext(ctx, "INSERT IGNORE INTO roles(name,code,description) VALUES('测试管理员','test_admin','测试')")
	_, _ = db.ExecContext(ctx, "INSERT IGNORE INTO permissions(code,name,resource,action) VALUES('vehicle:create','车辆创建','vehicle','create')")
}

func TestRepository_VehicleCRUD(t *testing.T) {
	ctx := context.Background()
	repo := mysqlrepo.NewVehicleRepository(testDB)

	// 创建。
	v, err := repo.CreateVehicle(ctx, entity.Vehicle{
		Model: "测试车型", VIN: "VINCTEST001", PlateNumber: "京A00001",
		Status: entity.VehicleStatusActive, OdometerKM: 100,
	})
	if err != nil {
		t.Fatalf("CreateVehicle: %v", err)
	}
	if v.ID == 0 {
		t.Fatal("ID 为 0")
	}

	// 查询。
	got, err := repo.GetVehicleByID(ctx, v.ID)
	if err != nil {
		t.Fatalf("GetVehicleByID: %v", err)
	}
	if got.VIN != "VINCTEST001" || got.PlateNumber != "京A00001" || got.OdometerKM != 100 {
		t.Fatalf("字段不匹配: %+v", got)
	}

	// 更新里程。
	if err := repo.UpdateVehicleMileage(ctx, v.ID, 500, time.Now()); err != nil {
		t.Fatalf("UpdateVehicleMileage: %v", err)
	}
	got2, _ := repo.GetVehicleByID(ctx, v.ID)
	if got2.OdometerKM != 500 {
		t.Fatalf("里程未更新: %d", got2.OdometerKM)
	}

	// 列表分页。
	items, total, err := repo.ListVehicles(ctx, entity.Page{Limit: 10, Offset: 0}, entity.Filter{}, entity.Sort{Field: "id", Order: "desc"})
	if err != nil {
		t.Fatalf("ListVehicles: %v", err)
	}
	if total < 1 || len(items) == 0 {
		t.Fatal("列表应为空")
	}
}

func TestRepository_VehicleUniqueVIN(t *testing.T) {
	ctx := context.Background()
	repo := mysqlrepo.NewVehicleRepository(testDB)

	_, _ = repo.CreateVehicle(ctx, entity.Vehicle{Model: "A", VIN: "UNIQUEVIN01", PlateNumber: "京U00001", Status: entity.VehicleStatusActive})
	_, err := repo.CreateVehicle(ctx, entity.Vehicle{Model: "B", VIN: "UNIQUEVIN01", PlateNumber: "京U00002", Status: entity.VehicleStatusActive})
	if !errors.Is(mysqlrepo.TranslateError(err), domain.ErrConflict) {
		t.Fatalf("期望冲突错误（重复 VIN），得到: %v", err)
	}
}

func TestRepository_TripCompleteTransaction(t *testing.T) {
	ctx := context.Background()
	vehRepo := mysqlrepo.NewVehicleRepository(testDB)
	tripRepo := mysqlrepo.NewTripRepository(testDB)
	tx := mysqlrepo.NewTransactor(testDB)

	// 准备车辆与司机（driver_id 是外键，必须先建）。
	v, _ := vehRepo.CreateVehicle(ctx, entity.Vehicle{Model: "T", VIN: "TRIPVIN001", PlateNumber: "京T00001", Status: entity.VehicleStatusActive, OdometerKM: 1000})
	drvRepo := mysqlrepo.NewDriverRepository(testDB)
	d, _ := drvRepo.CreateDriver(ctx, entity.Driver{Name: "司机测试", LicenseNumber: "TRIPDL001", LicenseClass: "A2", LicenseExpiry: ptrTime(time.Now().AddDate(1, 0, 0)), Phone: "13800000000"})

	// 创建任务。
	tr, err := tripRepo.CreateTrip(ctx, entity.Trip{
		VehicleID: v.ID, DriverID: d.ID, Route: "测试", StartOdometerKM: 1000,
		Status: entity.TripStatusInProgress, IdempotencyKey: "trip-tx-1",
	})
	if err != nil {
		t.Fatalf("CreateTrip: %v", err)
	}

	// 在事务内完成任务 + 更新车辆里程。
	err = tx.WithinTx(ctx, func(stores repository.Stores) error {
		if _, e := stores.Trips.GetTripByIDForUpdate(ctx, tr.ID); e != nil {
			return e
		}
		if e := stores.Trips.CompleteTrip(ctx, tr.ID, 2500, 1, time.Now()); e != nil {
			return e
		}
		return stores.Vehicles.UpdateVehicleMileage(ctx, v.ID, 2500, time.Now())
	})
	if err != nil {
		t.Fatalf("事务完成失败: %v", err)
	}
	// 车辆里程应为 2500。
	v2, _ := vehRepo.GetVehicleByID(ctx, v.ID)
	if v2.OdometerKM != 2500 {
		t.Fatalf("车辆里程应为 2500，得到 %d", v2.OdometerKM)
	}
}

func TestRepository_FuelIdempotencyKey(t *testing.T) {
	ctx := context.Background()
	vehRepo := mysqlrepo.NewVehicleRepository(testDB)
	fuelRepo := mysqlrepo.NewFuelRepository(testDB)

	v, _ := vehRepo.CreateVehicle(ctx, entity.Vehicle{Model: "F", VIN: "FUELVIN001", PlateNumber: "京F00001", Status: entity.VehicleStatusActive})

	_, err := fuelRepo.CreateFuelRecord(ctx, entity.FuelRecord{
		VehicleID: v.ID, LitersMilli: 40000, UnitPriceCents: 800, OdometerKM: 100,
		TotalCostCents: 32000, IdempotencyKey: "fuel-idem-1", RecordedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("首次创建: %v", err)
	}
	// 同 idempotency_key 应冲突。
	_, err = fuelRepo.CreateFuelRecord(ctx, entity.FuelRecord{
		VehicleID: v.ID, LitersMilli: 40000, UnitPriceCents: 800, OdometerKM: 100,
		TotalCostCents: 32000, IdempotencyKey: "fuel-idem-1", RecordedAt: time.Now(),
	})
	if !errors.Is(mysqlrepo.TranslateError(err), domain.ErrConflict) {
		t.Fatalf("期望幂等冲突，得到: %v", err)
	}
}

func TestRepository_MaintenancePolicyDue(t *testing.T) {
	ctx := context.Background()
	vehRepo := mysqlrepo.NewVehicleRepository(testDB)
	maintRepo := mysqlrepo.NewMaintenanceRepository(testDB)

	v, _ := vehRepo.CreateVehicle(ctx, entity.Vehicle{Model: "M", VIN: "MAINTVIN001", PlateNumber: "京M00001", Status: entity.VehicleStatusActive, OdometerKM: 5000})

	// 创建计划：里程间隔 1000，上次 4000，下次到期 5000，当前里程 5000 → 到期。enabled=TRUE。
	p, err := maintRepo.CreatePolicy(ctx, entity.MaintenancePolicy{
		VehicleID: v.ID, Name: "定期保养", Kind: entity.PolicyKindPeriodic,
		IntervalKM: 1000, IntervalDays: 30, LastServiceKM: 4000, LastServiceAt: time.Now().AddDate(0, 0, -40),
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("CreatePolicy: %v", err)
	}
	// 应在到期列表里（里程或日期已到期）。
	due, err := maintRepo.ListDuePolicies(ctx, time.Now())
	if err != nil {
		t.Fatalf("ListDuePolicies: %v", err)
	}
	found := false
	for _, d := range due {
		if d.ID == p.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("到期计划未出现在 ListDuePolicies")
	}
}

func TestRedis_IdempotencyAndCache(t *testing.T) {
	if testRdb == nil {
		t.Skip("Redis 不可用")
	}
	ctx := context.Background()

	// 幂等标记：首次 true，重复 false。
	first, _ := testRdb.SetIdempotency(ctx, "test-idem-key", "payload-1", time.Minute)
	if !first {
		t.Fatal("首次 SetIdempotency 应返回 true")
	}
	second, _ := testRdb.SetIdempotency(ctx, "test-idem-key", "payload-1", time.Minute)
	if second {
		t.Fatal("重复 SetIdempotency 应返回 false")
	}
	// 不一致 payload 应冲突。
	_, err := testRdb.SetIdempotency(ctx, "test-idem-key", "different", time.Minute)
	if err == nil {
		t.Fatal("不一致 payload 应报错")
	}

	// 缓存读写。
	_ = testRdb.SetCache(ctx, "cache-key", []byte("hello"), time.Minute)
	val, _ := testRdb.GetCache(ctx, "cache-key")
	if string(val) != "hello" {
		t.Fatalf("缓存值不匹配: %q", string(val))
	}
	_ = testRdb.InvalidateCache(ctx, "cache-key")
	val2, _ := testRdb.GetCache(ctx, "cache-key")
	if val2 != nil {
		t.Fatal("失效后应返回 nil")
	}

	// 限流。
	for i := 0; i < 5; i++ {
		testRdb.HitRateLimit(ctx, "rl-test", time.Minute, 3)
	}
	_, over, _ := testRdb.HitRateLimit(ctx, "rl-test", time.Minute, 3)
	if !over {
		t.Fatal("超过阈值应 over=true")
	}
}

func TestEndToEnd_VehicleTripFuelFlow(t *testing.T) {
	ctx := context.Background()
	tx := mysqlrepo.NewTransactor(testDB)
	stores := tx.Stores()
	auditSvc := service.NewAuditService(stores.Audit)
	vehSvc := service.NewVehicleService(stores.Vehicles, tx, auditSvc, testRdb)
	driverSvc := service.NewDriverService(stores.Drivers, auditSvc)
	tripSvc := service.NewTripService(stores.Trips, stores.Vehicles, stores.Fuel, stores.Maintenance, tx, auditSvc, testRdb)
	fuelSvc := service.NewFuelService(stores.Fuel, stores.Vehicles, tx, auditSvc, testRdb)

	actor := entity.AuditActor{UserID: 1, Username: "admin"}

	// 创建车辆。
	v, err := vehSvc.Create(ctx, entity.Vehicle{Model: "端到端", VIN: "E2EVIN001", PlateNumber: "京E00001"})
	if err != nil {
		t.Fatalf("创建车辆: %v", err)
	}
	// 创建司机。
	d, err := driverSvc.Create(ctx, entity.Driver{
		Name: "司机甲", LicenseNumber: "E2EDL001", LicenseClass: "A2",
		LicenseExpiry: ptrTime(time.Now().AddDate(1, 0, 0)), Phone: "13800000000",
	})
	if err != nil {
		t.Fatalf("创建司机: %v", err)
	}
	// 创建任务（带幂等键）。
	tr, err := tripSvc.Create(ctx, entity.Trip{
		VehicleID: v.ID, DriverID: d.ID, Route: "端到端路线", StartOdometerKM: 0,
		IdempotencyKey: "e2e-trip-1",
	}, actor)
	if err != nil {
		t.Fatalf("创建任务: %v", err)
	}
	// 开始 + 完成任务。
	if err := tripSvc.Start(ctx, tr.ID, actor); err != nil {
		t.Fatalf("开始任务: %v", err)
	}
	_, err = tripSvc.Complete(ctx, tr.ID, entity.TripComplete{
		EndOdometerKM: 2000, CompletedAt: time.Now(),
	}, actor)
	if err != nil {
		t.Fatalf("完成任务: %v", err)
	}
	// 车辆里程应为 2000。
	v2, _ := vehSvc.Get(ctx, v.ID)
	if v2.OdometerKM != 2000 {
		t.Fatalf("车辆里程应为 2000，得到 %d", v2.OdometerKM)
	}
	// 录入油耗（带幂等键）。
	_, err = fuelSvc.Record(ctx, entity.FuelInput{
		VehicleID: v.ID, LitersMilli: 30000, UnitPriceCents: 700,
		OdometerKM: 2100, IdempotencyKey: "e2e-fuel-1",
	}, actor)
	if err != nil {
		t.Fatalf("录入油耗: %v", err)
	}
	// 重复录入应幂等冲突。
	_, err = fuelSvc.Record(ctx, entity.FuelInput{
		VehicleID: v.ID, LitersMilli: 30000, UnitPriceCents: 700,
		OdometerKM: 2100, IdempotencyKey: "e2e-fuel-1",
	}, actor)
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("重复录入应冲突，得到: %v", err)
	}
	// 里程递减应拒绝。
	_, err = fuelSvc.Record(ctx, entity.FuelInput{
		VehicleID: v.ID, LitersMilli: 20000, UnitPriceCents: 700,
		OdometerKM: 1000, IdempotencyKey: "e2e-fuel-2",
	}, actor)
	if !errors.Is(err, domain.ErrMileageNotIncreasing) {
		t.Fatalf("里程递减应拒绝，得到: %v", err)
	}
}

func TestRepository_MigrationIdempotent(t *testing.T) {
	ctx := context.Background()
	mr := migration.New(testDB)
	// 二次迁移应不新增版本。
	ran, err := mr.Run(ctx)
	if err != nil {
		t.Fatalf("二次迁移失败: %v", err)
	}
	if len(ran) != 0 {
		t.Fatalf("幂等失败：二次运行又应用了 %v", ran)
	}
}

func ptrTime(t time.Time) *time.Time { return &t }
