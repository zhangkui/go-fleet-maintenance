package entity

import "time"

// Vehicle 车辆档案主实体。
type Vehicle struct {
	ID               int64      `json:"id"`
	Model            string     `json:"model"`
	VIN              string     `json:"vin"`
	PlateNumber      string     `json:"plate_number"`
	Status           string     `json:"status"`
	OdometerKM       int64      `json:"odometer_km"`
	Color            string     `json:"color"`
	EngineNo         string     `json:"engine_no"`
	PurchaseDate     *time.Time `json:"purchase_date,omitempty"`
	InsuranceExpiry  *time.Time `json:"insurance_expiry,omitempty"`
	InspectionExpiry *time.Time `json:"inspection_expiry,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// VehicleStatusHistory 车辆状态流转明细实体。
type VehicleStatusHistory struct {
	ID         int64     `json:"id"`
	VehicleID  int64     `json:"vehicle_id"`
	FromStatus string    `json:"from_status"`
	ToStatus   string    `json:"to_status"`
	Reason     string    `json:"reason"`
	ChangedBy  int64     `json:"changed_by"`
	ChangedAt  time.Time `json:"changed_at"`
}

// VehicleLicense 车辆证照实体（保险、年检等可到期项）。
type VehicleLicense struct {
	ID         int64      `json:"id"`
	VehicleID  int64      `json:"vehicle_id"`
	Kind       string     `json:"kind"` // insurance / inspection / road_transport / green_book
	Number     string     `json:"number"`
	IssueDate  *time.Time `json:"issue_date,omitempty"`
	ExpiryDate *time.Time `json:"expiry_date"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// 证照类型。
const (
	LicenseKindInsurance     = "insurance"
	LicenseKindInspection    = "inspection"
	LicenseKindRoadTransport = "road_transport"
	LicenseKindGreenBook     = "green_book"
)
