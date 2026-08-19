package mysqlrepo

import (
	"context"
	"database/sql"
	"time"

	"github.com/zhangkui/go-fleet-maintenance/internal/domain/entity"
)

// MaintenanceRepository 维保 MySQL 实现。
type MaintenanceRepository struct {
	db DBTX
}

// NewMaintenanceRepository 构造维保仓储。
func NewMaintenanceRepository(db DBTX) *MaintenanceRepository { return &MaintenanceRepository{db: db} }

// scanPolicy 扫描维保计划行。
func scanPolicy(sc func(...interface{}) error, p *entity.MaintenancePolicy) error {
	return sc(&p.ID, &p.VehicleID, &p.Name, &p.Kind, &p.IntervalKM, &p.IntervalDays,
		&p.LastServiceKM, &p.LastServiceAt, &p.NextDueKM, &p.NextDueAt, &p.Enabled, &p.CreatedAt, &p.UpdatedAt)
}

// CreatePolicy 创建维保计划，按里程与日期双条件计算首次到期。
func (r *MaintenanceRepository) CreatePolicy(ctx context.Context, p entity.MaintenancePolicy) (entity.MaintenancePolicy, error) {
	p.NextDueKM = p.LastServiceKM + p.IntervalKM
	if p.IntervalDays > 0 {
		p.NextDueAt = p.LastServiceAt.AddDate(0, 0, p.IntervalDays)
	} else {
		p.NextDueAt = p.LastServiceAt
	}
	res, err := r.db.ExecContext(ctx, `INSERT INTO maintenance_policies(vehicle_id,name,kind,interval_km,interval_days,last_service_km,last_service_at,next_due_km,next_due_at,enabled)
		VALUES(?,?,?,?,?,?,?,?,?,?)`, p.VehicleID, p.Name, p.Kind, p.IntervalKM, p.IntervalDays, p.LastServiceKM, p.LastServiceAt, p.NextDueKM, p.NextDueAt, p.Enabled)
	if err != nil {
		return entity.MaintenancePolicy{}, TranslateError(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return entity.MaintenancePolicy{}, err
	}
	p.ID = id
	p.CreatedAt = time.Now()
	p.UpdatedAt = p.CreatedAt
	return p, nil
}

// GetPolicyByID 按 ID 查维保计划。
func (r *MaintenanceRepository) GetPolicyByID(ctx context.Context, id int64) (entity.MaintenancePolicy, error) {
	var p entity.MaintenancePolicy
	err := scanPolicy(r.db.QueryRowContext(ctx, `SELECT id,vehicle_id,name,kind,interval_km,interval_days,last_service_km,last_service_at,next_due_km,next_due_at,enabled,created_at,updated_at
		FROM maintenance_policies WHERE id=?`, id).Scan, &p)
	if err != nil {
		return entity.MaintenancePolicy{}, TranslateError(err)
	}
	return p, nil
}

// ListPolicies 查车辆维保计划。
func (r *MaintenanceRepository) ListPolicies(ctx context.Context, vehicleID int64) ([]entity.MaintenancePolicy, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,vehicle_id,name,kind,interval_km,interval_days,last_service_km,last_service_at,next_due_km,next_due_at,enabled,created_at,updated_at
		FROM maintenance_policies WHERE vehicle_id=? ORDER BY id`, vehicleID)
	if err != nil {
		return nil, TranslateError(err)
	}
	defer rows.Close()
	var out []entity.MaintenancePolicy
	for rows.Next() {
		var p entity.MaintenancePolicy
		if err := scanPolicy(rows.Scan, &p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

// ListDuePolicies 列出已到期（里程或日期）且启用的维保计划。
// 里程到期：join vehicles 取车辆当前里程，next_due_km <= v.odometer_km。
// 日期到期：next_due_at <= now。
func (r *MaintenanceRepository) ListDuePolicies(ctx context.Context, now time.Time) ([]entity.MaintenancePolicy, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT p.id,p.vehicle_id,p.name,p.kind,p.interval_km,p.interval_days,p.last_service_km,p.last_service_at,p.next_due_km,p.next_due_at,p.enabled,p.created_at,p.updated_at
		FROM maintenance_policies p JOIN vehicles v ON v.id=p.vehicle_id
		WHERE p.enabled=TRUE AND (p.next_due_km>0 AND p.next_due_km<=v.odometer_km OR p.next_due_at<=?) ORDER BY p.id`,
		now.Format("2006-01-02"))
	if err != nil {
		return nil, TranslateError(err)
	}
	defer rows.Close()
	var out []entity.MaintenancePolicy
	for rows.Next() {
		var p entity.MaintenancePolicy
		if err := scanPolicy(rows.Scan, &p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

// UpdatePolicyLastService 更新计划上次保养里程与日期，并重算下次到期。
func (r *MaintenanceRepository) UpdatePolicyLastService(ctx context.Context, id int64, km int64, at time.Time) error {
	// 先取计划以重算 next_due。
	var p entity.MaintenancePolicy
	if err := scanPolicy(r.db.QueryRowContext(ctx, `SELECT id,vehicle_id,name,kind,interval_km,interval_days,last_service_km,last_service_at,next_due_km,next_due_at,enabled,created_at,updated_at
		FROM maintenance_policies WHERE id=? FOR UPDATE`, id).Scan, &p); err != nil {
		return TranslateError(err)
	}
	nextKM := km + p.IntervalKM
	var nextAt time.Time
	if p.IntervalDays > 0 {
		nextAt = at.AddDate(0, 0, p.IntervalDays)
	} else {
		nextAt = at
	}
	_, err := r.db.ExecContext(ctx, `UPDATE maintenance_policies SET last_service_km=?, last_service_at=?, next_due_km=?, next_due_at=?, updated_at=? WHERE id=?`,
		km, at, nextKM, nextAt, time.Now(), id)
	return TranslateError(err)
}

// scanOrder 扫描工单行。
func scanOrder(sc func(...interface{}) error, o *entity.MaintenanceOrder) error {
	var policyID sql.NullInt64
	var dtStart, dtEnd, completed sql.NullTime
	if err := sc(&o.ID, &o.VehicleID, &policyID, &o.Kind, &o.Title, &o.Status, &o.IdempotencyKey,
		&dtStart, &dtEnd, &o.PartsCostCents, &o.LaborCostCents, &o.TotalCostCents, &completed, &o.CreatedBy, &o.CreatedAt, &o.UpdatedAt); err != nil {
		return err
	}
	if policyID.Valid {
		v := policyID.Int64
		o.PolicyID = &v
	}
	o.DowntimeStart = nullTime(dtStart)
	o.DowntimeEnd = nullTime(dtEnd)
	o.CompletedAt = nullTime(completed)
	return nil
}

// CreateOrder 创建工单及配件明细，事务由上层保证。
func (r *MaintenanceRepository) CreateOrder(ctx context.Context, o entity.MaintenanceOrder, parts []entity.MaintenanceOrderPart) (entity.MaintenanceOrder, error) {
	res, err := r.db.ExecContext(ctx, `INSERT INTO maintenance_orders(vehicle_id,policy_id,kind,title,status,idempotency_key,downtime_start,downtime_end,parts_cost_cents,labor_cost_cents,total_cost_cents,created_by)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, o.VehicleID, i64PtrPtr(o.PolicyID), o.Kind, o.Title, o.Status, o.IdempotencyKey,
		timePtrPtr(o.DowntimeStart), timePtrPtr(o.DowntimeEnd), o.PartsCostCents, o.LaborCostCents, o.TotalCostCents, o.CreatedBy)
	if err != nil {
		return entity.MaintenanceOrder{}, TranslateError(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return entity.MaintenanceOrder{}, err
	}
	o.ID = id
	// 不写配件明细。
	o.PartsCostCents = 0
	o.TotalCostCents = o.LaborCostCents
	o.CreatedAt = time.Now()
	o.UpdatedAt = o.CreatedAt
	return o, nil
}

// GetOrderByID 按 ID 查工单。
func (r *MaintenanceRepository) GetOrderByID(ctx context.Context, id int64) (entity.MaintenanceOrder, error) {
	var o entity.MaintenanceOrder
	err := scanOrder(r.db.QueryRowContext(ctx, `SELECT id,vehicle_id,policy_id,kind,title,status,idempotency_key,downtime_start,downtime_end,parts_cost_cents,labor_cost_cents,total_cost_cents,completed_at,created_by,created_at,updated_at
		FROM maintenance_orders WHERE id=?`, id).Scan, &o)
	if err != nil {
		return entity.MaintenanceOrder{}, TranslateError(err)
	}
	return o, nil
}

// GetOrderByIDForUpdate 事务内加行锁查工单。
func (r *MaintenanceRepository) GetOrderByIDForUpdate(ctx context.Context, id int64) (entity.MaintenanceOrder, error) {
	var o entity.MaintenanceOrder
	err := scanOrder(r.db.QueryRowContext(ctx, `SELECT id,vehicle_id,policy_id,kind,title,status,idempotency_key,downtime_start,downtime_end,parts_cost_cents,labor_cost_cents,total_cost_cents,completed_at,created_by,created_at,updated_at
		FROM maintenance_orders WHERE id=? FOR UPDATE`, id).Scan, &o)
	if err != nil {
		return entity.MaintenanceOrder{}, TranslateError(err)
	}
	return o, nil
}

// orderSortWhitelist 工单排序白名单。
var orderSortWhitelist = map[string]string{
	"id": "id", "created": "created_at", "status": "status", "cost": "total_cost_cents",
}

// OrderSortFields 返回工单排序白名单。
func OrderSortFields() map[string]string { return orderSortWhitelist }

// ListOrders 分页过滤排序查询工单。
func (r *MaintenanceRepository) ListOrders(ctx context.Context, page entity.Page, filter entity.Filter, sort entity.Sort) ([]entity.MaintenanceOrder, int64, error) {
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
	if filter.Kind != "" {
		where += " AND kind=?"
		args = append(args, filter.Kind)
	}
	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM maintenance_orders "+where, args...).Scan(&total); err != nil {
		return nil, 0, TranslateError(err)
	}
	order := " ORDER BY id DESC"
	if sort.Field != "" {
		order = " ORDER BY " + sort.Field + " " + sort.Order
	}
	args = append(args, page.Limit, page.Offset)
	rows, err := r.db.QueryContext(ctx, `SELECT id,vehicle_id,policy_id,kind,title,status,idempotency_key,downtime_start,downtime_end,parts_cost_cents,labor_cost_cents,total_cost_cents,completed_at,created_by,created_at,updated_at
		FROM maintenance_orders `+where+order+` LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, 0, TranslateError(err)
	}
	defer rows.Close()
	var out []entity.MaintenanceOrder
	for rows.Next() {
		var o entity.MaintenanceOrder
		if err := scanOrder(rows.Scan, &o); err != nil {
			return nil, 0, err
		}
		out = append(out, o)
	}
	return out, total, nil
}

// UpdateOrderStatus 更新工单状态。
func (r *MaintenanceRepository) UpdateOrderStatus(ctx context.Context, id int64, status string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE maintenance_orders SET status=?, updated_at=? WHERE id=?", status, time.Now(), id)
	return TranslateError(err)
}

// CompleteOrder 完成工单，写停运结束与完成时间。
func (r *MaintenanceRepository) CompleteOrder(ctx context.Context, id int64, downtimeEnd time.Time) error {
	_, err := r.db.ExecContext(ctx, `UPDATE maintenance_orders SET status=?, downtime_end=?, completed_at=?, updated_at=? WHERE id=?`,
		entity.OrderStatusCompleted, downtimeEnd, downtimeEnd, time.Now(), id)
	return TranslateError(err)
}

// HasOpenOrderForPolicy 报告某计划是否已有未完成工单（用于幂等触发）。
func (r *MaintenanceRepository) HasOpenOrderForPolicy(ctx context.Context, policyID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM maintenance_orders WHERE policy_id=? AND status IN ('pending','approved'))`, policyID).Scan(&exists)
	return exists, TranslateError(err)
}

// AppendOrderStatusHistory 追加工单状态流转。
func (r *MaintenanceRepository) AppendOrderStatusHistory(ctx context.Context, h entity.MaintenanceOrderStatusHistory) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO maintenance_order_status_history(order_id,from_status,to_status,note,changed_by)
		VALUES(?,?,?,?,?)`, h.OrderID, h.FromStatus, h.ToStatus, h.Note, h.ChangedBy)
	return TranslateError(err)
}

// ListOrderParts 查工单配件明细。
func (r *MaintenanceRepository) ListOrderParts(ctx context.Context, orderID int64) ([]entity.MaintenanceOrderPart, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,order_id,part_id,quantity,unit_cost_cents,line_total_cents
		FROM maintenance_order_parts WHERE order_id=? ORDER BY id`, orderID)
	if err != nil {
		return nil, TranslateError(err)
	}
	defer rows.Close()
	var out []entity.MaintenanceOrderPart
	for rows.Next() {
		var p entity.MaintenanceOrderPart
		if err := rows.Scan(&p.ID, &p.OrderID, &p.PartID, &p.Quantity, &p.UnitCostCents, &p.LineTotalCents); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}
