package service_test

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/zhangkui/go-fleet-maintenance/internal/domain/entity"
	"github.com/zhangkui/go-fleet-maintenance/internal/repository"
	mysqlrepo "github.com/zhangkui/go-fleet-maintenance/internal/repository/mysql"
	"github.com/zhangkui/go-fleet-maintenance/internal/service"
	"github.com/zhangkui/go-fleet-maintenance/internal/transport/http/handler"
)

type maintenanceStub struct {
	repository.MaintenanceRepository
	order         entity.MaintenanceOrder
	parts         []entity.MaintenanceOrderPart
	due           []entity.MaintenancePolicy
	open          bool
	createdOrders int
	completedAt   time.Time
}

func (s *maintenanceStub) CreateOrder(_ context.Context, order entity.MaintenanceOrder, parts []entity.MaintenanceOrderPart) (entity.MaintenanceOrder, error) {
	s.order, s.parts = order, parts
	s.createdOrders++
	order.ID = int64(s.createdOrders)
	return order, nil
}
func (s *maintenanceStub) GetOrderByIDForUpdate(context.Context, int64) (entity.MaintenanceOrder, error) {
	return s.order, nil
}
func (s *maintenanceStub) CompleteOrder(_ context.Context, _ int64, at time.Time) error {
	s.completedAt = at
	return nil
}
func (s *maintenanceStub) AppendOrderStatusHistory(context.Context, entity.MaintenanceOrderStatusHistory) error {
	return nil
}
func (s *maintenanceStub) HasOpenOrderForPolicy(context.Context, int64) (bool, error) {
	return s.open, nil
}
func (s *maintenanceStub) ListDuePolicies(context.Context, time.Time) ([]entity.MaintenancePolicy, error) {
	return s.due, nil
}
func (s *maintenanceStub) UpdatePolicyLastService(context.Context, int64, int64, time.Time) error {
	return nil
}

type partBehaviorStub struct {
	repository.PartRepository
	part     entity.Part
	delta    int64
	balance  int64
	movement entity.PartStockMovement
	low      []entity.Part
}

func (s *partBehaviorStub) GetPartByIDForUpdate(context.Context, int64) (entity.Part, error) {
	return s.part, nil
}
func (s *partBehaviorStub) UpdateStock(_ context.Context, _ int64, delta, balance int64) error {
	s.delta, s.balance = delta, balance
	return nil
}
func (s *partBehaviorStub) AppendStockMovement(_ context.Context, movement entity.PartStockMovement) error {
	s.movement = movement
	return nil
}
func (s *partBehaviorStub) ListLowStock(context.Context) ([]entity.Part, error) { return s.low, nil }

type reminderStub struct {
	repository.ReminderRepository
	pending       []entity.Reminder
	created       []entity.Reminder
	queriedBefore time.Time
}

func (s *reminderStub) ListPendingReminders(_ context.Context, before time.Time) ([]entity.Reminder, error) {
	s.queriedBefore = before
	return s.pending, nil
}
func (s *reminderStub) CreateReminder(_ context.Context, reminder entity.Reminder) (entity.Reminder, error) {
	s.created = append(s.created, reminder)
	return reminder, nil
}

type expiringVehicleStub struct {
	repository.VehicleRepository
	items    []entity.Vehicle
	from, to time.Time
}

func (s *expiringVehicleStub) ListExpiringDocuments(_ context.Context, from, to time.Time) ([]entity.Vehicle, error) {
	s.from, s.to = from, to
	return s.items, nil
}

type expiringDriverStub struct{ repository.DriverRepository }

func (s *expiringDriverStub) ListExpiringLicenses(context.Context, time.Time, time.Time) ([]entity.Driver, error) {
	return nil, nil
}

func TestBug018_MaintenancePartsConsumption(t *testing.T) {
	maint := &maintenanceStub{}
	parts := &partBehaviorStub{part: entity.Part{ID: 7, SKU: "P7", StockQuantity: 10}}
	vehicles := &vehicleStub{vehicle: entity.Vehicle{ID: 3, Status: entity.VehicleStatusActive}}
	tx := transactorStub{stores: repository.Stores{Maintenance: maint, Parts: parts, Vehicles: vehicles}}
	svc := service.NewMaintenanceService(maint, vehicles, parts, tx, service.NewAuditService(&auditCaptureStub{}), nil)
	created, err := svc.CreateOrder(context.Background(), entity.MaintenanceOrder{VehicleID: 3, Title: "service", IdempotencyKey: "bug018", LaborCostCents: 500}, []entity.MaintenanceOrderPart{{PartID: 7, Quantity: 3, UnitCostCents: 200}}, entity.AuditActor{UserID: 9})
	if err != nil {
		t.Fatal(err)
	}
	if parts.balance != 7 || parts.delta != -3 || parts.movement.BalanceAfter != 7 {
		t.Fatalf("stock delta=%d balance=%d movement=%+v", parts.delta, parts.balance, parts.movement)
	}
	if created.ID == 0 {
		t.Fatal("order not created")
	}

	db, mock, _ := sqlmock.New()
	defer db.Close()
	mock.ExpectExec("INSERT INTO maintenance_orders").WillReturnResult(sqlmock.NewResult(11, 1))
	mock.ExpectExec("INSERT INTO maintenance_order_parts").WithArgs(int64(11), int64(7), int64(3), int64(200), int64(600)).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("UPDATE maintenance_orders SET parts_cost_cents=\\?, total_cost_cents=\\? WHERE id=\\?").WithArgs(int64(600), int64(1100), int64(11)).WillReturnResult(sqlmock.NewResult(0, 1))
	order, err := mysqlrepo.NewMaintenanceRepository(db).CreateOrder(context.Background(), entity.MaintenanceOrder{VehicleID: 3, Kind: entity.PolicyKindRepair, Title: "service", Status: entity.OrderStatusPending, LaborCostCents: 500, IdempotencyKey: "bug018", CreatedBy: 9}, []entity.MaintenanceOrderPart{{PartID: 7, Quantity: 3, UnitCostCents: 200}})
	if err != nil {
		t.Fatal(err)
	}
	if order.PartsCostCents != 600 || order.TotalCostCents != 1100 {
		t.Fatalf("costs=%+v", order)
	}
}

func TestBug019_MaintenanceVehicleRecovery(t *testing.T) {
	maint := &maintenanceStub{order: entity.MaintenanceOrder{ID: 1, VehicleID: 3, Status: entity.OrderStatusInProgress}}
	vehicles := &vehicleStub{vehicle: entity.Vehicle{ID: 3, Status: entity.VehicleStatusInMaintenance}}
	tx := transactorStub{stores: repository.Stores{Maintenance: maint, Vehicles: vehicles}}
	svc := service.NewMaintenanceService(maint, vehicles, nil, tx, service.NewAuditService(&auditCaptureStub{}), nil)
	end := time.Now()
	if err := svc.CompleteOrder(context.Background(), 1, entity.OrderComplete{DowntimeEnd: end}, entity.AuditActor{UserID: 8}); err != nil {
		t.Fatal(err)
	}
	if vehicles.status != entity.VehicleStatusActive || vehicles.history.ToStatus != entity.VehicleStatusActive {
		t.Fatalf("status=%q history=%+v", vehicles.status, vehicles.history)
	}
	db, mock, _ := sqlmock.New()
	defer db.Close()
	mock.ExpectExec("UPDATE maintenance_orders SET status=\\?, downtime_end=\\?, completed_at=\\?, updated_at=\\? WHERE id=\\?").WithArgs(entity.OrderStatusCompleted, end, end, sqlmock.AnyArg(), int64(1)).WillReturnResult(sqlmock.NewResult(0, 1))
	if err := mysqlrepo.NewMaintenanceRepository(db).CompleteOrder(context.Background(), 1, end); err != nil {
		t.Fatal(err)
	}
}

func TestBug020_MaintenancePolicyDedup(t *testing.T) {
	policy := entity.MaintenancePolicy{ID: 4, VehicleID: 3, Name: "periodic", Kind: entity.PolicyKindPeriodic}
	maint := &maintenanceStub{due: []entity.MaintenancePolicy{policy}, open: true}
	vehicles := &vehicleStub{vehicle: entity.Vehicle{ID: 3, Status: entity.VehicleStatusActive}}
	tx := transactorStub{stores: repository.Stores{Maintenance: maint, Vehicles: vehicles}}
	svc := service.NewMaintenanceService(maint, vehicles, nil, tx, service.NewAuditService(&auditCaptureStub{}), nil)
	count, err := svc.TriggerDue(context.Background(), entity.AuditActor{UserID: 1})
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 || maint.createdOrders != 0 {
		t.Fatalf("count=%d orders=%d", count, maint.createdOrders)
	}
	db, mock, _ := sqlmock.New()
	defer db.Close()
	mock.ExpectQuery("status IN \\('pending','approved','in_progress'\\)").WithArgs(int64(4)).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	if _, err := mysqlrepo.NewMaintenanceRepository(db).HasOpenOrderForPolicy(context.Background(), 4); err != nil {
		t.Fatal(err)
	}
}

func TestBug021_PartStockFloor(t *testing.T) {
	parts := &partBehaviorStub{part: entity.Part{ID: 1, StockQuantity: 5}}
	tx := transactorStub{stores: repository.Stores{Parts: parts}}
	svc := service.NewPartService(parts, tx, service.NewAuditService(&auditCaptureStub{}))
	result, err := svc.AdjustStock(context.Background(), entity.PartAdjust{PartID: 1, Change: -5}, entity.AuditActor{UserID: 2})
	if err != nil {
		t.Fatalf("zero balance should be allowed: %v", err)
	}
	if result.StockQuantity != 0 || parts.balance != 0 {
		t.Fatalf("result=%d stored=%d", result.StockQuantity, parts.balance)
	}
	db, mock, _ := sqlmock.New()
	defer db.Close()
	mock.ExpectExec("UPDATE parts SET stock_quantity=\\?, updated_at=\\? WHERE id=\\?").WithArgs(int64(0), sqlmock.AnyArg(), int64(1)).WillReturnResult(sqlmock.NewResult(0, 1))
	if err := mysqlrepo.NewPartRepository(db).UpdateStock(context.Background(), 1, -5, 0); err != nil {
		t.Fatal(err)
	}
}

func TestBug022_PartReorderBoundary(t *testing.T) {
	parts := &partBehaviorStub{low: []entity.Part{{ID: 1, StockQuantity: 5, ReorderPoint: 5}}}
	result, err := service.NewPartService(parts, nil, nil).ListLowStock(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Fatalf("boundary item omitted: %+v", result)
	}
	db, mock, _ := sqlmock.New()
	defer db.Close()
	mock.ExpectQuery("stock_quantity<=reorder_point").WillReturnRows(sqlmock.NewRows([]string{"id", "sku", "name", "unit", "stock_quantity", "reorder_point", "unit_cost_cents", "created_at", "updated_at"}))
	if _, err := mysqlrepo.NewPartRepository(db).ListLowStock(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestBug023_ReminderDeduplication(t *testing.T) {
	due := time.Now().AddDate(0, 0, 10)
	repo := &reminderStub{pending: []entity.Reminder{{EntityType: entity.ReminderVehicleInsurance, EntityID: 1, DueAt: due}}}
	vehicles := &expiringVehicleStub{items: []entity.Vehicle{{ID: 1, PlateNumber: "A", InsuranceExpiry: &due}}}
	svc := service.NewReminderService(repo, vehicles, &expiringDriverStub{}, &maintenanceStub{}, nil)
	result, err := svc.Scan(context.Background(), 30)
	if err != nil {
		t.Fatal(err)
	}
	if result.Created != 0 || len(repo.created) != 0 {
		t.Fatalf("result=%+v created=%d", result, len(repo.created))
	}
	if repo.queriedBefore.Before(due.Add(-time.Second)) {
		t.Fatalf("pending cutoff=%v due=%v", repo.queriedBefore, due)
	}
	db, mock, _ := sqlmock.New()
	defer db.Close()
	mock.ExpectQuery("status='pending' AND due_at<=\\?").WithArgs(due).WillReturnRows(sqlmock.NewRows([]string{"id", "entity_type", "entity_id", "due_at", "message", "status", "created_at", "sent_at"}))
	if _, err := mysqlrepo.NewReminderRepository(db).ListPendingReminders(context.Background(), due); err != nil {
		t.Fatal(err)
	}
}

func TestBug024_ReminderScanHorizon(t *testing.T) {
	vehicles := &expiringVehicleStub{}
	svc := service.NewReminderService(&reminderStub{}, vehicles, &expiringDriverStub{}, &maintenanceStub{}, nil)
	if _, err := svc.Scan(context.Background(), 0); err != nil {
		t.Fatal(err)
	}
	days := int(vehicles.to.Sub(vehicles.from).Hours() / 24)
	if days < 29 || days > 31 {
		t.Fatalf("default horizon=%d days", days)
	}
	requestVehicles := &expiringVehicleStub{}
	requestService := service.NewReminderService(&reminderStub{}, requestVehicles, &expiringDriverStub{}, &maintenanceStub{}, nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("POST", "/api/reminders/scan?days=7", nil)
	handler.NewReminderHandler(requestService, 1024).Scan(recorder, request)
	requestDays := int(requestVehicles.to.Sub(requestVehicles.from).Hours() / 24)
	if requestDays < 6 || requestDays > 8 {
		t.Fatalf("days query ignored: horizon=%d", requestDays)
	}
}
