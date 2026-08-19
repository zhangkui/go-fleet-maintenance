package mysqlrepo

import (
	"context"
	"time"

	"github.com/zhangkui/go-fleet-maintenance/internal/domain/entity"
)

// ReportRepository 报表只读 MySQL 实现。
type ReportRepository struct {
	db DBTX
}

// NewReportRepository 构造报表仓储。
func NewReportRepository(db DBTX) *ReportRepository { return &ReportRepository{db: db} }

// FleetSummary 汇总车队总览。
func (r *ReportRepository) FleetSummary(ctx context.Context) (entity.FleetSummary, error) {
	var s entity.FleetSummary
	s.GeneratedAt = time.Now()
	queries := []struct {
		sql  string
		dst  *int64
		args []interface{}
	}{
		{"SELECT COUNT(*) FROM vehicles", &s.TotalVehicles, nil},
		{"SELECT COUNT(*) FROM vehicles WHERE status='active'", &s.ActiveVehicles, nil},
		{"SELECT COUNT(*) FROM vehicles WHERE status='in_maintenance'", &s.InMaintenance, nil},
		{"SELECT COUNT(*) FROM drivers", &s.TotalDrivers, nil},
		{"SELECT COUNT(*) FROM drivers WHERE status='active'", &s.ActiveDrivers, nil},
		{"SELECT COUNT(*) FROM trips WHERE status IN ('scheduled','in_progress')", &s.OpenTrips, nil},
		{"SELECT COUNT(*) FROM maintenance_orders WHERE status IN ('pending','approved','in_progress')", &s.OpenOrders, nil},
		{"SELECT COUNT(*) FROM parts WHERE stock_quantity<=reorder_point", &s.LowStockParts, nil},
	}
	for _, q := range queries {
		if err := r.db.QueryRowContext(ctx, q.sql, q.args...).Scan(q.dst); err != nil {
			return s, TranslateError(err)
		}
	}
	// 到期证照数（30 天内）。
	from := time.Now()
	to := from.AddDate(0, 0, 30)
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM vehicles WHERE insurance_expiry BETWEEN ? AND ? OR inspection_expiry BETWEEN ? AND ?`,
		from.Format("2006-01-02"), to.Format("2006-01-02"), from.Format("2006-01-02"), to.Format("2006-01-02")).Scan(&s.ExpiringDocuments); err != nil {
		return s, TranslateError(err)
	}
	return s, nil
}

// VehicleUtilization 车辆利用率报表。
func (r *ReportRepository) VehicleUtilization(ctx context.Context, from, to time.Time, page entity.Page) ([]entity.VehicleUtilization, int64, error) {
	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM vehicles").Scan(&total); err != nil {
		return nil, 0, TranslateError(err)
	}
	rows, err := r.db.QueryContext(ctx, `SELECT v.id,v.plate_number,v.model,
		COUNT(t.id), COALESCE(SUM(t.end_odometer_km - t.start_odometer_km),0), COALESCE(SUM(f.liters_milli),0)
		FROM vehicles v
		LEFT JOIN trips t ON t.vehicle_id=v.id AND t.status='completed' AND t.completed_at BETWEEN ? AND ?
		LEFT JOIN fuel_records f ON f.vehicle_id=v.id AND f.recorded_at BETWEEN ? AND ?
		GROUP BY v.id ORDER BY v.id LIMIT ? OFFSET ?`,
		from, to, from, to, page.Limit, page.Offset)
	if err != nil {
		return nil, 0, TranslateError(err)
	}
	defer rows.Close()
	var out []entity.VehicleUtilization
	for rows.Next() {
		var u entity.VehicleUtilization
		if err := rows.Scan(&u.VehicleID, &u.PlateNumber, &u.Model, &u.TripCount, &u.TotalDistance, &u.TotalFuelLiters); err != nil {
			return nil, 0, err
		}
		if u.TotalDistance > 0 {
			u.UtilizationPct = float64(u.TripCount) / 30.0 * 100
		}
		out = append(out, u)
	}
	return out, total, nil
}

// FuelEfficiency 油耗效率报表。
func (r *ReportRepository) FuelEfficiency(ctx context.Context, from, to time.Time, page entity.Page) ([]entity.FuelEfficiencyReport, int64, error) {
	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(DISTINCT vehicle_id) FROM fuel_records WHERE recorded_at BETWEEN ? AND ?", from, to).Scan(&total); err != nil {
		return nil, 0, TranslateError(err)
	}
	rows, err := r.db.QueryContext(ctx, `SELECT v.id,v.plate_number,
		COALESCE(SUM(f.liters_milli),0), COALESCE(MAX(f.odometer_km)-MIN(f.odometer_km),0), COALESCE(SUM(f.total_cost_cents),0),
		COUNT(CASE WHEN f.abnormal=TRUE THEN 1 END)
		FROM vehicles v JOIN fuel_records f ON f.vehicle_id=v.id AND f.recorded_at BETWEEN ? AND ?
		GROUP BY v.id, v.plate_number ORDER BY v.id LIMIT ? OFFSET ?`,
		from, to, page.Limit, page.Offset)
	if err != nil {
		return nil, 0, TranslateError(err)
	}
	defer rows.Close()
	var out []entity.FuelEfficiencyReport
	for rows.Next() {
		var rep entity.FuelEfficiencyReport
		if err := rows.Scan(&rep.VehicleID, &rep.PlateNumber, &rep.TotalLiters, &rep.TotalDistanceKM, &rep.TotalCostCents, &rep.AbnormalRecords); err != nil {
			return nil, 0, err
		}
		if rep.TotalDistanceKM > 0 {
			rep.LitersPer100KM = float64(rep.TotalLiters) / 1000.0 / float64(rep.TotalDistanceKM) * 100
			rep.CostPerKM = rep.TotalCostCents / rep.TotalDistanceKM
		}
		out = append(out, rep)
	}
	return out, total, nil
}

// MaintenanceCost 维保成本报表。
func (r *ReportRepository) MaintenanceCost(ctx context.Context, from, to time.Time, page entity.Page) ([]entity.MaintenanceCostReport, int64, error) {
	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(DISTINCT vehicle_id) FROM maintenance_orders WHERE created_at BETWEEN ? AND ?", from, to).Scan(&total); err != nil {
		return nil, 0, TranslateError(err)
	}
	rows, err := r.db.QueryContext(ctx, `SELECT v.id,v.plate_number,
		COUNT(o.id), COALESCE(SUM(o.parts_cost_cents),0), COALESCE(SUM(o.labor_cost_cents),0), COALESCE(SUM(o.total_cost_cents),0),
		COALESCE(SUM(CASE WHEN o.downtime_end IS NOT NULL AND o.downtime_start IS NOT NULL THEN TIMESTAMPDIFF(HOUR,o.downtime_start,o.downtime_end) ELSE 0 END),0)
		FROM vehicles v JOIN maintenance_orders o ON o.vehicle_id=v.id AND o.created_at BETWEEN ? AND ?
		GROUP BY v.id, v.plate_number ORDER BY v.id LIMIT ? OFFSET ?`,
		from, to, page.Limit, page.Offset)
	if err != nil {
		return nil, 0, TranslateError(err)
	}
	defer rows.Close()
	var out []entity.MaintenanceCostReport
	for rows.Next() {
		var rep entity.MaintenanceCostReport
		if err := rows.Scan(&rep.VehicleID, &rep.PlateNumber, &rep.OrderCount, &rep.PartsCostCents, &rep.LaborCostCents, &rep.TotalCostCents, &rep.DowntimeHours); err != nil {
			return nil, 0, err
		}
		out = append(out, rep)
	}
	return out, total, nil
}
