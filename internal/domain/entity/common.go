// Package entity 定义全部领域实体、值对象和业务枚举。
package entity

import "time"

// Money 金额统一使用最小货币单位（分），禁止 float64。
type Money int64

// Quantity 数量统一使用整数最小单位，例如毫升、克、件。
type Quantity int64

// Page 分页参数。
type Page struct {
	Limit  int
	Offset int
}

// PageResult 通用分页结果包装。
type PageResult[T any] struct {
	Items []T   `json:"items"`
	Total int64 `json:"total"`
	Limit int   `json:"limit"`
	Page  int   `json:"page"`
}

// Sort 白名单排序参数。
type Sort struct {
	Field string
	Order string // asc 或 desc
}

// Filter 通用过滤字段集合，具体子模块按需取用。
type Filter struct {
	Status    string
	Keyword   string
	VehicleID int64
	DriverID  int64
	From      time.Time
	To        time.Time
	Abnormal  *bool
	Kind      string
}

// AuditActor 操作发起者上下文，由认证中间件注入。
type AuditActor struct {
	UserID   int64
	Username string
	Role     string
	IP       string
}

// 车辆状态。
const (
	VehicleStatusActive        = "active"
	VehicleStatusInMaintenance = "in_maintenance"
	VehicleStatusRetired       = "retired"
)

// 车辆状态合法流转图。
var vehicleTransitions = map[string]map[string]bool{
	VehicleStatusActive:        {VehicleStatusInMaintenance: true, VehicleStatusRetired: true},
	VehicleStatusInMaintenance: {VehicleStatusActive: true, VehicleStatusRetired: true},
	VehicleStatusRetired:       {},
}

// CanTransitionVehicle 报告车辆状态能否按合法顺序流转。
func CanTransitionVehicle(from, to string) bool {
	if from == to {
		return true
	}
	allowed, ok := vehicleTransitions[from]
	if !ok {
		return false
	}
	return allowed[to]
}

// 任务状态。
const (
	TripStatusScheduled  = "scheduled"
	TripStatusInProgress = "in_progress"
	TripStatusCompleted  = "completed"
	TripStatusCancelled  = "cancelled"
)

var tripTransitions = map[string]map[string]bool{
	TripStatusScheduled:  {TripStatusInProgress: true, TripStatusCancelled: true},
	TripStatusInProgress: {TripStatusCompleted: true, TripStatusCancelled: true},
	TripStatusCompleted:  {},
	TripStatusCancelled:  {},
}

// CanTransitionTrip 报告任务状态能否按合法顺序流转。
func CanTransitionTrip(from, to string) bool {
	if from == to {
		return true
	}
	allowed, ok := tripTransitions[from]
	if !ok {
		return false
	}
	return allowed[to]
}

// 维保工单状态。
const (
	OrderStatusPending    = "pending"
	OrderStatusApproved   = "approved"
	OrderStatusInProgress = "in_progress"
	OrderStatusCompleted  = "completed"
	OrderStatusCancelled  = "cancelled"
)

var orderTransitions = map[string]map[string]bool{
	OrderStatusPending:    {OrderStatusApproved: true, OrderStatusCancelled: true},
	OrderStatusApproved:   {OrderStatusInProgress: true, OrderStatusCancelled: true},
	OrderStatusInProgress: {OrderStatusCompleted: true, OrderStatusCancelled: true},
	OrderStatusCompleted:  {},
	OrderStatusCancelled:  {},
}

// CanTransitionOrder 报告维保工单状态能否按合法顺序流转。
func CanTransitionOrder(from, to string) bool {
	if from == to {
		return true
	}
	allowed, ok := orderTransitions[from]
	if !ok {
		return false
	}
	return allowed[to]
}

// 违章状态。
const (
	ViolationStatusPending   = "pending"
	ViolationStatusPaid      = "paid"
	ViolationStatusContested = "contested"
)

// 库存变动原因。
const (
	StockReasonPurchase         = "purchase"
	StockReasonOrderConsumption = "order_consumption"
	StockReasonAdjustment       = "adjustment"
	StockReasonReturn           = "return"
)

// 提醒类型。
const (
	ReminderVehicleInsurance  = "vehicle_insurance"
	ReminderVehicleInspection = "vehicle_inspection"
	ReminderDriverLicense     = "driver_license"
	ReminderMaintenancePolicy = "maintenance_policy"
)

// 提醒状态。
const (
	ReminderStatusPending   = "pending"
	ReminderStatusSent      = "sent"
	ReminderStatusDismissed = "dismissed"
)

// 工具函数：bool 指针。
func BoolPtr(b bool) *bool { return &b }

// IDReq 通用 ID 请求字段。
type IDReq struct {
	ID int64 `json:"id"`
}
