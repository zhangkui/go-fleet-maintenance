package mysqlrepo

import (
	"context"
	"database/sql"
	"time"

	"github.com/zhangkui/go-fleet-maintenance/internal/domain/entity"
)

// TripRepository 出车任务 MySQL 实现。
type TripRepository struct {
	db DBTX
}

// NewTripRepository 构造任务仓储。
func NewTripRepository(db DBTX) *TripRepository { return &TripRepository{db: db} }

// scanTrip 扫描任务行的公共逻辑。
func scanTrip(sc func(...interface{}) error, t *entity.Trip) error {
	var endOdo sql.NullInt64
	var completedAt sql.NullTime
	var completedBy sql.NullInt64
	if err := sc(&t.ID, &t.VehicleID, &t.DriverID, &t.Route, &t.LoadKG, &t.StartOdometerKM,
		&endOdo, &t.Status, &t.IdempotencyKey, &completedAt, &completedBy, &t.Note, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return err
	}
	if endOdo.Valid {
		v := endOdo.Int64
		t.EndOdometerKM = &v
	}
	t.CompletedAt = nullTime(completedAt)
	if completedBy.Valid {
		v := completedBy.Int64
		t.CompletedBy = &v
	}
	return nil
}

// CreateTrip 创建任务，幂等键唯一约束兜底。
func (r *TripRepository) CreateTrip(ctx context.Context, t entity.Trip) (entity.Trip, error) {
	res, err := r.db.ExecContext(ctx, `INSERT INTO trips(vehicle_id,driver_id,route,load_kg,start_odometer_km,status,idempotency_key,note)
		VALUES(?,?,?,?,?,?,?,?)`, t.VehicleID, t.DriverID, t.Route, t.LoadKG, t.StartOdometerKM, t.Status, t.IdempotencyKey, t.Note)
	if err != nil {
		return entity.Trip{}, TranslateError(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return entity.Trip{}, err
	}
	t.ID = id
	t.CreatedAt = time.Now()
	t.UpdatedAt = t.CreatedAt
	return t, nil
}

// GetTripByID 按 ID 查任务。
func (r *TripRepository) GetTripByID(ctx context.Context, id int64) (entity.Trip, error) {
	var t entity.Trip
	err := scanTrip(r.db.QueryRowContext(ctx, `SELECT id,vehicle_id,driver_id,route,load_kg,start_odometer_km,end_odometer_km,status,idempotency_key,completed_at,completed_by,note,created_at,updated_at
		FROM trips WHERE id=?`, id).Scan, &t)
	if err != nil {
		return entity.Trip{}, TranslateError(err)
	}
	return t, nil
}

// GetTripByIDForUpdate 事务内加行锁查任务。
func (r *TripRepository) GetTripByIDForUpdate(ctx context.Context, id int64) (entity.Trip, error) {
	var t entity.Trip
	err := scanTrip(r.db.QueryRowContext(ctx, `SELECT id,vehicle_id,driver_id,route,load_kg,start_odometer_km,end_odometer_km,status,idempotency_key,completed_at,completed_by,note,created_at,updated_at
		FROM trips WHERE id=? FOR UPDATE`, id).Scan, &t)
	if err != nil {
		return entity.Trip{}, TranslateError(err)
	}
	return t, nil
}

// tripSortWhitelist 任务排序白名单。
var tripSortWhitelist = map[string]string{
	"id": "id", "created": "created_at", "start": "start_odometer_km", "status": "status",
}

// TripSortFields 返回任务排序白名单。
func TripSortFields() map[string]string { return tripSortWhitelist }

// ListTrips 分页、过滤、白名单排序查询任务。
func (r *TripRepository) ListTrips(ctx context.Context, page entity.Page, filter entity.Filter, sort entity.Sort) ([]entity.Trip, int64, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	if filter.Status != "" {
		where += " AND status=?"
		args = append(args, filter.Status)
	}
	if filter.VehicleID != 0 {
		where += " AND vehicle_id=?"
		args = append(args, filter.VehicleID)
	}
	if filter.DriverID != 0 {
		where += " AND driver_id=?"
		args = append(args, filter.DriverID)
	}
	if !filter.From.IsZero() {
		where += " AND created_at>=?"
		args = append(args, filter.From)
	}
	if !filter.To.IsZero() {
		where += " AND created_at<=?"
		args = append(args, filter.To)
	}
	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM trips "+where, args...).Scan(&total); err != nil {
		return nil, 0, TranslateError(err)
	}
	order := " ORDER BY id DESC"
	if sort.Field != "" {
		order = " ORDER BY " + sort.Field + " " + sort.Order
	}
	args = append(args, page.Limit, page.Offset)
	rows, err := r.db.QueryContext(ctx, `SELECT id,vehicle_id,driver_id,route,load_kg,start_odometer_km,end_odometer_km,status,idempotency_key,completed_at,completed_by,note,created_at,updated_at
		FROM trips `+where+order+` LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, 0, TranslateError(err)
	}
	defer rows.Close()
	var out []entity.Trip
	for rows.Next() {
		var t entity.Trip
		if err := scanTrip(rows.Scan, &t); err != nil {
			return nil, 0, err
		}
		out = append(out, t)
	}
	return out, total, nil
}

// StartTrip 开始任务（scheduled -> in_progress）。
func (r *TripRepository) StartTrip(ctx context.Context, id int64, by int64, at time.Time) error {
	_, err := r.db.ExecContext(ctx, "UPDATE trips SET status=?, completed_at=?, updated_at=? WHERE id=?", entity.TripStatusInProgress, at, at, id)
	return TranslateError(err)
}

// CompleteTrip 完成任务，写结束里程与完成时间。
func (r *TripRepository) CompleteTrip(ctx context.Context, id int64, endOdometer int64, completedBy int64, at time.Time) error {
	_, err := r.db.ExecContext(ctx, `UPDATE trips SET status=?, end_odometer_km=?, completed_at=?, completed_by=?, updated_at=? WHERE id=?`,
		entity.TripStatusCompleted, endOdometer, at, completedBy, at, id)
	return TranslateError(err)
}

// CancelTrip 取消任务。
func (r *TripRepository) CancelTrip(ctx context.Context, id int64, by int64, at time.Time) error {
	_, err := r.db.ExecContext(ctx, "UPDATE trips SET status=?, updated_at=? WHERE id=?", entity.TripStatusCancelled, at, id)
	return TranslateError(err)
}

// AppendTripStatusHistory 追加任务状态流转。
func (r *TripRepository) AppendTripStatusHistory(ctx context.Context, h entity.TripStatusHistory) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO trip_status_history(trip_id,from_status,to_status,note,changed_by)
		VALUES(?,?,?,?,?)`, h.TripID, h.FromStatus, h.ToStatus, h.Note, h.ChangedBy)
	return TranslateError(err)
}

// CreateHandover 创建任务交接。
func (r *TripRepository) CreateHandover(ctx context.Context, h entity.TripHandover) (entity.TripHandover, error) {
	res, err := r.db.ExecContext(ctx, `INSERT INTO trip_handovers(trip_id,from_driver_id,to_driver_id,handover_at,location,note)
		VALUES(?,?,?,?,?,?)`, h.TripID, h.FromDriverID, h.ToDriverID, h.HandoverAt, h.Location, h.Note)
	if err != nil {
		return entity.TripHandover{}, TranslateError(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return entity.TripHandover{}, err
	}
	h.ID = id
	h.CreatedAt = time.Now()
	return h, nil
}
