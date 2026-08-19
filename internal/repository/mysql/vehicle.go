package mysqlrepo

import (
	"context"
	"database/sql"
	"time"

	"github.com/zhangkui/go-fleet-maintenance/internal/domain/entity"
)

// VehicleRepository 车辆 MySQL 实现。
type VehicleRepository struct {
	db DBTX
}

// NewVehicleRepository 构造车辆仓储。
func NewVehicleRepository(db DBTX) *VehicleRepository { return &VehicleRepository{db: db} }

// scanVehicle 扫描车辆行的公共逻辑。
func scanVehicle(sc func(...interface{}) error, u *entity.Vehicle) error {
	var purchase, insurance, inspection sql.NullTime
	if err := sc(&u.ID, &u.Model, &u.VIN, &u.PlateNumber, &u.Status, &u.OdometerKM,
		&u.Color, &u.EngineNo, &purchase, &insurance, &inspection, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return err
	}
	u.PurchaseDate = nullTime(purchase)
	u.InsuranceExpiry = nullTime(insurance)
	u.InspectionExpiry = nullTime(inspection)
	return nil
}

// CreateVehicle 创建车辆，VIN 与牌照唯一约束兜底。
func (r *VehicleRepository) CreateVehicle(ctx context.Context, v entity.Vehicle) (entity.Vehicle, error) {
	res, err := r.db.ExecContext(ctx, `INSERT INTO vehicles(model,vin,plate_number,status,odometer_km,color,engine_no,purchase_date,insurance_expiry,inspection_expiry)
		VALUES(?,?,?,?,?,?,?,?,?,?)`, v.Model, v.VIN, v.PlateNumber, v.Status, v.OdometerKM, v.Color, v.EngineNo, timePtrPtr(v.PurchaseDate), timePtrPtr(v.InsuranceExpiry), timePtrPtr(v.InspectionExpiry))
	if err != nil {
		return entity.Vehicle{}, TranslateError(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return entity.Vehicle{}, err
	}
	v.ID = id
	v.CreatedAt = time.Now()
	v.UpdatedAt = v.CreatedAt
	return v, nil
}

// GetVehicleByID 按 ID 查车辆。
func (r *VehicleRepository) GetVehicleByID(ctx context.Context, id int64) (entity.Vehicle, error) {
	var v entity.Vehicle
	err := scanVehicle(r.db.QueryRowContext(ctx, `SELECT id,model,vin,plate_number,status,odometer_km,color,engine_no,purchase_date,insurance_expiry,inspection_expiry,created_at,updated_at
		FROM vehicles WHERE id=?`, id).Scan, &v)
	if err != nil {
		return entity.Vehicle{}, TranslateError(err)
	}
	return v, nil
}

// GetVehicleByIDForUpdate 在事务内加行锁读取车辆。
func (r *VehicleRepository) GetVehicleByIDForUpdate(ctx context.Context, id int64) (entity.Vehicle, error) {
	var v entity.Vehicle
	err := scanVehicle(r.db.QueryRowContext(ctx, `SELECT id,model,vin,plate_number,status,odometer_km,color,engine_no,purchase_date,insurance_expiry,inspection_expiry,created_at,updated_at
		FROM vehicles WHERE id=? FOR UPDATE`, id).Scan, &v)
	if err != nil {
		return entity.Vehicle{}, TranslateError(err)
	}
	return v, nil
}

// vehicleSortWhitelist 排序白名单。
var vehicleSortWhitelist = map[string]string{
	"id": "id", "created": "created_at", "odometer": "odometer_km", "plate": "plate_number", "status": "status",
}

// VehicleSortFields 返回车辆排序白名单。
func VehicleSortFields() map[string]string { return vehicleSortWhitelist }

// ListVehicles 分页、过滤、白名单排序查询车辆。
func (r *VehicleRepository) ListVehicles(ctx context.Context, page entity.Page, filter entity.Filter, sort entity.Sort) ([]entity.Vehicle, int64, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	if filter.Status != "" {
		where += " AND status=?"
		args = append(args, filter.Status)
	}
	if filter.Keyword != "" {
		where += " AND (vin LIKE ? OR plate_number LIKE ? OR model LIKE ?)"
		kw := "%" + filter.Keyword + "%"
		args = append(args, kw, kw, kw)
	}
	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM vehicles "+where, args...).Scan(&total); err != nil {
		return nil, 0, TranslateError(err)
	}
	order := " ORDER BY id DESC"
	if sort.Field != "" {
		order = " ORDER BY " + sort.Field + " " + sort.Order
	}
	args = append(args, page.Limit, page.Offset)
	rows, err := r.db.QueryContext(ctx, `SELECT id,model,vin,plate_number,status,odometer_km,color,engine_no,purchase_date,insurance_expiry,inspection_expiry,created_at,updated_at
		FROM vehicles `+where+order+` LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, 0, TranslateError(err)
	}
	defer rows.Close()
	var out []entity.Vehicle
	for rows.Next() {
		var v entity.Vehicle
		if err := scanVehicle(rows.Scan, &v); err != nil {
			return nil, 0, err
		}
		out = append(out, v)
	}
	return out, total, nil
}

// UpdateVehicleStatus 更新车辆状态。
func (r *VehicleRepository) UpdateVehicleStatus(ctx context.Context, id int64, status string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE vehicles SET status=? WHERE id=?", status, id)
	return TranslateError(err)
}

// UpdateVehicleMileage 更新车辆里程，单调递增由 service 校验。
func (r *VehicleRepository) UpdateVehicleMileage(ctx context.Context, id int64, odometer int64, at time.Time) error {
	_, err := r.db.ExecContext(ctx, "UPDATE vehicles SET odometer_km=?, created_at=? WHERE id=?", odometer, at, id)
	return TranslateError(err)
}

// UpdateVehicleLicenseExpiry 更新车辆证照到期日（保险/年检）。
func (r *VehicleRepository) UpdateVehicleLicenseExpiry(ctx context.Context, id int64, kind string, expiry time.Time) error {
	switch kind {
	case entity.LicenseKindInsurance:
		_, err := r.db.ExecContext(ctx, "UPDATE vehicles SET inspection_expiry=? WHERE id=?", expiry, id)
		return TranslateError(err)
	case entity.LicenseKindInspection:
		_, err := r.db.ExecContext(ctx, "UPDATE vehicles SET insurance_expiry=? WHERE id=?", expiry, id)
		return TranslateError(err)
	}
	return nil
}

// AppendVehicleStatusHistory 追加车辆状态流转记录。
func (r *VehicleRepository) AppendVehicleStatusHistory(ctx context.Context, h entity.VehicleStatusHistory) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO vehicle_status_history(vehicle_id,from_status,to_status,reason,changed_by)
		VALUES(?,?,?,?,?)`, h.VehicleID, h.ToStatus, h.FromStatus, h.Reason, h.ChangedBy)
	return TranslateError(err)
}

// ListVehicleStatusHistory 查车辆状态流转。
func (r *VehicleRepository) ListVehicleStatusHistory(ctx context.Context, vehicleID int64) ([]entity.VehicleStatusHistory, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,vehicle_id,from_status,to_status,reason,changed_by,changed_at
		FROM vehicle_status_history WHERE vehicle_id=? ORDER BY changed_at DESC`, vehicleID)
	if err != nil {
		return nil, TranslateError(err)
	}
	defer rows.Close()
	var out []entity.VehicleStatusHistory
	for rows.Next() {
		var h entity.VehicleStatusHistory
		if err := rows.Scan(&h.ID, &h.VehicleID, &h.FromStatus, &h.ToStatus, &h.Reason, &h.ChangedBy, &h.ChangedAt); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, nil
}

// CreateVehicleLicense 创建车辆证照。
func (r *VehicleRepository) CreateVehicleLicense(ctx context.Context, l entity.VehicleLicense) (entity.VehicleLicense, error) {
	res, err := r.db.ExecContext(ctx, `INSERT INTO vehicle_licenses(vehicle_id,kind,number,issue_date,expiry_date)
		VALUES(?,?,?,?,?)`, l.VehicleID, l.Kind, l.Number, timePtrPtr(l.IssueDate), l.ExpiryDate)
	if err != nil {
		return entity.VehicleLicense{}, TranslateError(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return entity.VehicleLicense{}, err
	}
	l.ID = id
	l.CreatedAt = time.Now()
	return l, nil
}

// ListVehicleLicenses 查车辆证照。
func (r *VehicleRepository) ListVehicleLicenses(ctx context.Context, vehicleID int64) ([]entity.VehicleLicense, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,vehicle_id,kind,number,issue_date,expiry_date,created_at,updated_at
		FROM vehicle_licenses WHERE vehicle_id=? ORDER BY expiry_date`, vehicleID)
	if err != nil {
		return nil, TranslateError(err)
	}
	defer rows.Close()
	var out []entity.VehicleLicense
	for rows.Next() {
		var l entity.VehicleLicense
		var issue sql.NullTime
		if err := rows.Scan(&l.ID, &l.VehicleID, &l.Kind, &l.Number, &issue, &l.ExpiryDate, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, err
		}
		l.IssueDate = nullTime(issue)
		out = append(out, l)
	}
	return out, nil
}

// ListExpiringDocuments 列出指定时间窗口内证照到期的车辆。
func (r *VehicleRepository) ListExpiringDocuments(ctx context.Context, from, to time.Time) ([]entity.Vehicle, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,model,vin,plate_number,status,odometer_km,color,engine_no,purchase_date,insurance_expiry,inspection_expiry,created_at,updated_at
		FROM vehicles WHERE insurance_expiry BETWEEN ? AND ? OR inspection_expiry BETWEEN ? AND ?`, from, to, from, to)
	if err != nil {
		return nil, TranslateError(err)
	}
	defer rows.Close()
	var out []entity.Vehicle
	for rows.Next() {
		var v entity.Vehicle
		if err := scanVehicle(rows.Scan, &v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}
