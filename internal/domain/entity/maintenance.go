package entity

import "time"

// MaintenancePolicy 维保周期计划实体，里程+日期双条件触发。
type MaintenancePolicy struct {
	ID            int64     `json:"id"`
	VehicleID     int64     `json:"vehicle_id"`
	Name          string    `json:"name"`
	Kind          string    `json:"kind"` // periodic / inspection / repair
	IntervalKM    int64     `json:"interval_km"`
	IntervalDays  int       `json:"interval_days"`
	LastServiceKM int64     `json:"last_service_km"`
	LastServiceAt time.Time `json:"last_service_at"`
	NextDueKM     int64     `json:"next_due_km"`
	NextDueAt     time.Time `json:"next_due_at"`
	Enabled       bool      `json:"enabled"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// 维保类型。
const (
	PolicyKindPeriodic   = "periodic"
	PolicyKindInspection = "inspection"
	PolicyKindRepair     = "repair"
)

// MaintenanceOrder 维保工单主实体。
type MaintenanceOrder struct {
	ID             int64      `json:"id"`
	VehicleID      int64      `json:"vehicle_id"`
	PolicyID       *int64     `json:"policy_id,omitempty"`
	Kind           string     `json:"kind"`
	Title          string     `json:"title"`
	Status         string     `json:"status"`
	IdempotencyKey string     `json:"idempotency_key,omitempty"`
	DowntimeStart  *time.Time `json:"downtime_start,omitempty"`
	DowntimeEnd    *time.Time `json:"downtime_end,omitempty"`
	PartsCostCents int64      `json:"parts_cost_cents"`
	LaborCostCents int64      `json:"labor_cost_cents"`
	TotalCostCents int64      `json:"total_cost_cents"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	CreatedBy      int64      `json:"created_by"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// MaintenanceOrderPart 工单配件明细实体。
type MaintenanceOrderPart struct {
	ID             int64 `json:"id"`
	OrderID        int64 `json:"order_id"`
	PartID         int64 `json:"part_id"`
	Quantity       int64 `json:"quantity"`
	UnitCostCents  int64 `json:"unit_cost_cents"`
	LineTotalCents int64 `json:"line_total_cents"`
}

// MaintenanceOrderStatusHistory 工单状态流转明细实体。
type MaintenanceOrderStatusHistory struct {
	ID         int64     `json:"id"`
	OrderID    int64     `json:"order_id"`
	FromStatus string    `json:"from_status"`
	ToStatus   string    `json:"to_status"`
	Note       string    `json:"note"`
	ChangedBy  int64     `json:"changed_by"`
	ChangedAt  time.Time `json:"changed_at"`
}

// OrderComplete 完成工单请求。
type OrderComplete struct {
	DowntimeEnd time.Time `json:"downtime_end"`
	Note        string    `json:"note"`
}
