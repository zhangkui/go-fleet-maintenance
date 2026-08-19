package entity

import "time"

// Driver 司机主实体。
type Driver struct {
	ID            int64      `json:"id"`
	Name          string     `json:"name"`
	LicenseNumber string     `json:"license_number"`
	LicenseClass  string     `json:"license_class"`
	LicenseExpiry *time.Time `json:"license_expiry"`
	Phone         string     `json:"phone"`
	Status        string     `json:"status"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// 司机状态。
const (
	DriverStatusActive    = "active"
	DriverStatusSuspended = "suspended"
	DriverStatusResigned  = "resigned"
)

// DriverVehicleBinding 司机-车辆绑定多对多关联实体。
type DriverVehicleBinding struct {
	ID        int64      `json:"id"`
	DriverID  int64      `json:"driver_id"`
	VehicleID int64      `json:"vehicle_id"`
	StartDate time.Time  `json:"start_date"`
	EndDate   *time.Time `json:"end_date,omitempty"`
	Status    string     `json:"status"` // active / ended
	CreatedAt time.Time  `json:"created_at"`
}

// 绑定状态。
const (
	BindingStatusActive = "active"
	BindingStatusEnded  = "ended"
)

// DriverSchedule 司机排班实体。
type DriverSchedule struct {
	ID        int64     `json:"id"`
	DriverID  int64     `json:"driver_id"`
	ShiftDate time.Time `json:"shift_date"`
	ShiftType string    `json:"shift_type"` // morning / evening / night / off
	Note      string    `json:"note"`
	CreatedAt time.Time `json:"created_at"`
}

// 班次类型。
const (
	ShiftMorning = "morning"
	ShiftEvening = "evening"
	ShiftNight   = "night"
	ShiftOff     = "rest"
)

// DriverViolation 司机违章记录实体。
type DriverViolation struct {
	ID            int64     `json:"id"`
	DriverID      int64     `json:"driver_id"`
	OccurredAt    time.Time `json:"occurred_at"`
	ViolationType string    `json:"violation_type"`
	Points        int       `json:"points"`
	FineCents     int64     `json:"fine_cents"`
	Description   string    `json:"description"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}
