package entity

import "time"

// FuelRecord 油耗记录实体，同时作为车辆加油数量与金额流水。
type FuelRecord struct {
	ID             int64     `json:"id"`
	VehicleID      int64     `json:"vehicle_id"`
	LitersMilli    int64     `json:"liters_milli"`     // 毫升，整数最小单位
	UnitPriceCents int64     `json:"unit_price_cents"` // 元/升精确到分
	OdometerKM     int64     `json:"odometer_km"`
	TotalCostCents int64     `json:"total_cost_cents"` // 分
	Abnormal       bool      `json:"abnormal"`
	RecordedAt     time.Time `json:"recorded_at"`
	IdempotencyKey string    `json:"idempotency_key,omitempty"`
	CreatedBy      int64     `json:"created_by"`
	CreatedAt      time.Time `json:"created_at"`
}

// FuelInput 油耗录入请求，支持显式总价或由单价×升数计算。
type FuelInput struct {
	VehicleID      int64     `json:"vehicle_id"`
	LitersMilli    int64     `json:"liters_milli"`
	UnitPriceCents int64     `json:"unit_price_cents"`
	OdometerKM     int64     `json:"odometer_km"`
	TotalCostCents int64     `json:"total_cost_cents"`
	RecordedAt     time.Time `json:"recorded_at"`
	IdempotencyKey string    `json:"idempotency_key"`
}
