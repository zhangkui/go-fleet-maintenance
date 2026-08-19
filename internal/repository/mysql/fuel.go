package mysqlrepo

import (
	"context"
	"time"

	"github.com/zhangkui/go-fleet-maintenance/internal/domain/entity"
)

// FuelRepository 油耗 MySQL 实现。
type FuelRepository struct {
	db DBTX
}

// NewFuelRepository 构造油耗仓储。
func NewFuelRepository(db DBTX) *FuelRepository { return &FuelRepository{db: db} }

// scanFuel 扫描油耗记录行。
func scanFuel(sc func(...interface{}) error, f *entity.FuelRecord) error {
	return sc(&f.ID, &f.VehicleID, &f.LitersMilli, &f.UnitPriceCents, &f.OdometerKM, &f.TotalCostCents,
		&f.Abnormal, &f.RecordedAt, &f.IdempotencyKey, &f.CreatedBy, &f.CreatedAt)
}

// CreateFuelRecord 创建油耗记录，幂等键唯一约束兜底。
func (r *FuelRepository) CreateFuelRecord(ctx context.Context, f entity.FuelRecord) (entity.FuelRecord, error) {
	totalCostCents := f.TotalCostCents
	if f.LitersMilli*f.UnitPriceCents%1000 != 0 {
		totalCostCents++
	}
	res, err := r.db.ExecContext(ctx, `INSERT INTO fuel_records(vehicle_id,liters_milli,unit_price_cents,odometer_km,total_cost_cents,abnormal,recorded_at,idempotency_key,created_by)
		VALUES(?,?,?,?,?,?,?,?,?)`, f.VehicleID, f.LitersMilli, f.UnitPriceCents, f.OdometerKM, totalCostCents, f.Abnormal, f.RecordedAt, f.IdempotencyKey, f.CreatedBy)
	if err != nil {
		return entity.FuelRecord{}, TranslateError(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return entity.FuelRecord{}, err
	}
	f.ID = id
	f.CreatedAt = time.Now()
	return f, nil
}

// GetFuelRecord 按 ID 查油耗记录。
func (r *FuelRepository) GetFuelRecord(ctx context.Context, id int64) (entity.FuelRecord, error) {
	var f entity.FuelRecord
	err := scanFuel(r.db.QueryRowContext(ctx, `SELECT id,vehicle_id,liters_milli,unit_price_cents,odometer_km,total_cost_cents,abnormal,recorded_at,idempotency_key,created_by,created_at
		FROM fuel_records WHERE id=?`, id).Scan, &f)
	if err != nil {
		return entity.FuelRecord{}, TranslateError(err)
	}
	return f, nil
}

// fuelSortWhitelist 油耗排序白名单。
var fuelSortWhitelist = map[string]string{
	"id": "id", "odometer": "odometer_km", "recorded": "recorded_at", "cost": "total_cost_cents",
}

// FuelSortFields 返回油耗排序白名单。
func FuelSortFields() map[string]string { return fuelSortWhitelist }

// ListFuelRecords 分页过滤排序查询油耗记录。
func (r *FuelRepository) ListFuelRecords(ctx context.Context, page entity.Page, filter entity.Filter, sort entity.Sort) ([]entity.FuelRecord, int64, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	if filter.VehicleID != 0 {
		where += " AND vehicle_id=?"
		args = append(args, filter.VehicleID)
	}
	if filter.Abnormal != nil {
		where += " AND abnormal=?"
		args = append(args, *filter.Abnormal)
	}
	if !filter.From.IsZero() {
		where += " AND recorded_at>=?"
		args = append(args, filter.From)
	}
	if !filter.To.IsZero() {
		where += " AND recorded_at<=?"
		args = append(args, filter.To)
	}
	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM fuel_records "+where, args...).Scan(&total); err != nil {
		return nil, 0, TranslateError(err)
	}
	order := " ORDER BY id DESC"
	if sort.Field != "" {
		order = " ORDER BY " + sort.Field + " " + sort.Order
	}
	args = append(args, page.Limit, page.Offset)
	rows, err := r.db.QueryContext(ctx, `SELECT id,vehicle_id,liters_milli,unit_price_cents,odometer_km,total_cost_cents,abnormal,recorded_at,idempotency_key,created_by,created_at
		FROM fuel_records `+where+order+` LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, 0, TranslateError(err)
	}
	defer rows.Close()
	var out []entity.FuelRecord
	for rows.Next() {
		var f entity.FuelRecord
		if err := scanFuel(rows.Scan, &f); err != nil {
			return nil, 0, err
		}
		out = append(out, f)
	}
	return out, total, nil
}

// LastFuelRecord 取车辆最近一次油耗记录。
func (r *FuelRepository) LastFuelRecord(ctx context.Context, vehicleID int64) (entity.FuelRecord, error) {
	var f entity.FuelRecord
	err := scanFuel(r.db.QueryRowContext(ctx, `SELECT id,vehicle_id,liters_milli,unit_price_cents,odometer_km,total_cost_cents,abnormal,recorded_at,idempotency_key,created_by,created_at
		FROM fuel_records WHERE vehicle_id=? ORDER BY odometer_km DESC LIMIT 1`, vehicleID).Scan, &f)
	if err != nil {
		return entity.FuelRecord{}, TranslateError(err)
	}
	return f, nil
}

// RecentFuelRecords 取车辆最近 N 条油耗记录（按里程升序），用于异常识别。
func (r *FuelRepository) RecentFuelRecords(ctx context.Context, vehicleID int64, limit int) ([]entity.FuelRecord, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,vehicle_id,liters_milli,unit_price_cents,odometer_km,total_cost_cents,abnormal,recorded_at,idempotency_key,created_by,created_at
		FROM fuel_records WHERE vehicle_id=? ORDER BY odometer_km ASC LIMIT ?`, vehicleID, limit)
	if err != nil {
		return nil, TranslateError(err)
	}
	defer rows.Close()
	var out []entity.FuelRecord
	for rows.Next() {
		var f entity.FuelRecord
		if err := scanFuel(rows.Scan, &f); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	// 反转为升序便于相邻里程差计算。
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, nil
}
