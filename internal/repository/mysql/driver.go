package mysqlrepo

import (
	"context"
	"database/sql"
	"time"

	"github.com/zhangkui/go-fleet-maintenance/internal/domain/entity"
)

// DriverRepository 司机 MySQL 实现。
type DriverRepository struct {
	db DBTX
}

// NewDriverRepository 构造司机仓储。
func NewDriverRepository(db DBTX) *DriverRepository { return &DriverRepository{db: db} }

// CreateDriver 创建司机。
func (r *DriverRepository) CreateDriver(ctx context.Context, d entity.Driver) (entity.Driver, error) {
	res, err := r.db.ExecContext(ctx, `INSERT INTO drivers(name,license_number,license_class,license_expiry,phone,status)
		VALUES(?,?,?,?,?,?)`, d.Name, d.LicenseNumber, d.LicenseClass, dateOnlyPtr(d.LicenseExpiry), d.Phone, entity.DriverStatusActive)
	if err != nil {
		return entity.Driver{}, TranslateError(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return entity.Driver{}, err
	}
	d.ID = id
	d.Status = entity.DriverStatusActive
	d.CreatedAt = time.Now()
	d.UpdatedAt = d.CreatedAt
	return d, nil
}

// GetDriverByID 按 ID 查司机。
func (r *DriverRepository) GetDriverByID(ctx context.Context, id int64) (entity.Driver, error) {
	var d entity.Driver
	var lic sql.NullTime
	err := r.db.QueryRowContext(ctx, `SELECT id,name,license_number,license_class,license_expiry,phone,status,created_at,updated_at
		FROM drivers WHERE id=?`, id).Scan(&d.ID, &d.Name, &d.LicenseNumber, &d.LicenseClass, &lic, &d.Phone, &d.Status, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return entity.Driver{}, TranslateError(err)
	}
	d.LicenseExpiry = nullTime(lic)
	return d, nil
}

// ListDrivers 分页过滤查询司机。
func (r *DriverRepository) ListDrivers(ctx context.Context, page entity.Page, filter entity.Filter) ([]entity.Driver, int64, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	if filter.Status != "" {
		where += " AND status=?"
		args = append(args, filter.Status)
	}
	if filter.Keyword != "" {
		where += " AND (name LIKE ? OR license_number LIKE ? OR phone LIKE ?)"
		kw := "%" + filter.Keyword + "%"
		args = append(args, kw, kw, kw)
	}
	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM drivers "+where, args...).Scan(&total); err != nil {
		return nil, 0, TranslateError(err)
	}
	args = append(args, page.Limit, page.Offset)
	rows, err := r.db.QueryContext(ctx, `SELECT id,name,license_number,license_class,license_expiry,phone,status,created_at,updated_at
		FROM drivers `+where+` ORDER BY id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, 0, TranslateError(err)
	}
	defer rows.Close()
	var out []entity.Driver
	for rows.Next() {
		var d entity.Driver
		var lic sql.NullTime
		if err := rows.Scan(&d.ID, &d.Name, &d.LicenseNumber, &d.LicenseClass, &lic, &d.Phone, &d.Status, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, 0, err
		}
		d.LicenseExpiry = nullTime(lic)
		out = append(out, d)
	}
	return out, total, nil
}

// UpdateDriverStatus 更新司机状态。
func (r *DriverRepository) UpdateDriverStatus(ctx context.Context, id int64, status string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE drivers SET status=? WHERE id=?", status, id)
	return TranslateError(err)
}

// CreateBinding 创建司机-车辆绑定。
func (r *DriverRepository) CreateBinding(ctx context.Context, b entity.DriverVehicleBinding) (entity.DriverVehicleBinding, error) {
	res, err := r.db.ExecContext(ctx, `INSERT INTO driver_vehicle_bindings(driver_id,vehicle_id,start_date,end_date,status)
		VALUES(?,?,?,?,?)`, b.DriverID, b.VehicleID, b.StartDate, dateOnlyPtr(b.EndDate), b.Status)
	if err != nil {
		return entity.DriverVehicleBinding{}, TranslateError(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return entity.DriverVehicleBinding{}, err
	}
	b.ID = id
	b.CreatedAt = time.Now()
	return b, nil
}

// ListActiveBindings 列出某车辆在指定时间生效的绑定。
func (r *DriverRepository) ListActiveBindings(ctx context.Context, vehicleID int64, at time.Time) ([]entity.DriverVehicleBinding, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,driver_id,vehicle_id,start_date,end_date,status,created_at
		FROM driver_vehicle_bindings WHERE vehicle_id=? AND status=? AND start_date<=? AND (end_date IS NULL OR end_date>=?)`,
		vehicleID, entity.BindingStatusActive, at.Format("2006-01-02"), at.Format("2006-01-02"))
	if err != nil {
		return nil, TranslateError(err)
	}
	defer rows.Close()
	var out []entity.DriverVehicleBinding
	for rows.Next() {
		var b entity.DriverVehicleBinding
		var end sql.NullTime
		if err := rows.Scan(&b.ID, &b.DriverID, &b.VehicleID, &b.StartDate, &end, &b.Status, &b.CreatedAt); err != nil {
			return nil, err
		}
		b.EndDate = nullTime(end)
		out = append(out, b)
	}
	return out, nil
}

// HasActiveBindingForVehicle 报告某车辆在指定时间是否有活跃绑定。
func (r *DriverRepository) HasActiveBindingForVehicle(ctx context.Context, vehicleID int64, at time.Time) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM driver_vehicle_bindings
		WHERE vehicle_id=? AND status=? AND start_date<=? AND (end_date IS NULL OR end_date>=?))`,
		vehicleID, entity.BindingStatusActive, at.Format("2006-01-02"), at.Format("2006-01-02")).Scan(&exists)
	return exists, TranslateError(err)
}

// CreateSchedule 创建排班。
func (r *DriverRepository) CreateSchedule(ctx context.Context, s entity.DriverSchedule) (entity.DriverSchedule, error) {
	res, err := r.db.ExecContext(ctx, `INSERT INTO driver_schedules(driver_id,shift_date,shift_type,note)
		VALUES(?,?,?,?)`, s.DriverID, s.ShiftDate.Format("2006-01-02"), s.ShiftType, s.Note)
	if err != nil {
		return entity.DriverSchedule{}, TranslateError(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return entity.DriverSchedule{}, err
	}
	s.ID = id
	s.CreatedAt = time.Now()
	return s, nil
}

// ListScheduleByDriver 按司机与时间范围列排班。
func (r *DriverRepository) ListScheduleByDriver(ctx context.Context, driverID int64, from, to time.Time) ([]entity.DriverSchedule, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,driver_id,shift_date,shift_type,note,created_at
		FROM driver_schedules WHERE driver_id=? AND shift_date BETWEEN ? AND ? ORDER BY shift_date`,
		driverID, from.Format("2006-01-02"), to.Format("2006-01-02"))
	if err != nil {
		return nil, TranslateError(err)
	}
	defer rows.Close()
	var out []entity.DriverSchedule
	for rows.Next() {
		var s entity.DriverSchedule
		if err := rows.Scan(&s.ID, &s.DriverID, &s.ShiftDate, &s.ShiftType, &s.Note, &s.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

// CreateViolation 创建违章记录。
func (r *DriverRepository) CreateViolation(ctx context.Context, v entity.DriverViolation) (entity.DriverViolation, error) {
	res, err := r.db.ExecContext(ctx, `INSERT INTO driver_violations(driver_id,occurred_at,violation_type,points,fine_cents,description,status)
		VALUES(?,?,?,?,?,?,?)`, v.DriverID, v.OccurredAt, v.ViolationType, v.Points, v.FineCents, v.Description, entity.ViolationStatusPending)
	if err != nil {
		return entity.DriverViolation{}, TranslateError(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return entity.DriverViolation{}, err
	}
	v.ID = id
	v.Status = entity.ViolationStatusPending
	v.CreatedAt = time.Now()
	return v, nil
}

// ListViolations 查司机违章。
func (r *DriverRepository) ListViolations(ctx context.Context, driverID int64) ([]entity.DriverViolation, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,driver_id,occurred_at,violation_type,points,fine_cents,description,status,created_at
		FROM driver_violations WHERE driver_id=? ORDER BY occurred_at DESC`, driverID)
	if err != nil {
		return nil, TranslateError(err)
	}
	defer rows.Close()
	var out []entity.DriverViolation
	for rows.Next() {
		var v entity.DriverViolation
		if err := rows.Scan(&v.ID, &v.DriverID, &v.OccurredAt, &v.ViolationType, &v.Points, &v.FineCents, &v.Description, &v.Status, &v.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// ListExpiringLicenses 列出指定窗口内驾照到期的司机。
func (r *DriverRepository) ListExpiringLicenses(ctx context.Context, from, to time.Time) ([]entity.Driver, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,name,license_number,license_class,license_expiry,phone,status,created_at,updated_at
		FROM drivers WHERE license_expiry BETWEEN ? AND ?`, from.Format("2006-01-02"), to.Format("2006-01-02"))
	if err != nil {
		return nil, TranslateError(err)
	}
	defer rows.Close()
	var out []entity.Driver
	for rows.Next() {
		var d entity.Driver
		var lic sql.NullTime
		if err := rows.Scan(&d.ID, &d.Name, &d.LicenseNumber, &d.LicenseClass, &lic, &d.Phone, &d.Status, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		d.LicenseExpiry = nullTime(lic)
		out = append(out, d)
	}
	return out, nil
}
