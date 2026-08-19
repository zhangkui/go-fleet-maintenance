package entity

import "time"

// Trip 出车任务主实体。
type Trip struct {
	ID              int64      `json:"id"`
	VehicleID       int64      `json:"vehicle_id"`
	DriverID        int64      `json:"driver_id"`
	Route           string     `json:"route"`
	LoadKG          int64      `json:"load_kg"`
	StartOdometerKM int64      `json:"start_odometer_km"`
	EndOdometerKM   *int64     `json:"end_odometer_km,omitempty"`
	Status          string     `json:"status"`
	IdempotencyKey  string     `json:"idempotency_key,omitempty"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	CompletedBy     *int64     `json:"completed_by,omitempty"`
	Note            string     `json:"note"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// TripStatusHistory 任务状态流转明细实体。
type TripStatusHistory struct {
	ID         int64     `json:"id"`
	TripID     int64     `json:"trip_id"`
	FromStatus string    `json:"from_status"`
	ToStatus   string    `json:"to_status"`
	Note       string    `json:"note"`
	ChangedBy  int64     `json:"changed_by"`
	ChangedAt  time.Time `json:"changed_at"`
}

// TripHandover 任务交接明细实体。
type TripHandover struct {
	ID           int64     `json:"id"`
	TripID       int64     `json:"trip_id"`
	FromDriverID int64     `json:"from_driver_id"`
	ToDriverID   int64     `json:"to_driver_id"`
	HandoverAt   time.Time `json:"handover_at"`
	Location     string    `json:"location"`
	Note         string    `json:"note"`
	CreatedAt    time.Time `json:"created_at"`
}

// TripComplete 完成任务的请求体。
type TripComplete struct {
	EndOdometerKM int64     `json:"end_odometer_km"`
	CompletedAt   time.Time `json:"completed_at"`
	Note          string    `json:"note"`
}
