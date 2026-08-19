package entity

import "time"

// FleetSummary 车队总览报表。
type FleetSummary struct {
	TotalVehicles     int64     `json:"total_vehicles"`
	ActiveVehicles    int64     `json:"active_vehicles"`
	InMaintenance     int64     `json:"in_maintenance"`
	TotalDrivers      int64     `json:"total_drivers"`
	ActiveDrivers     int64     `json:"active_drivers"`
	OpenTrips         int64     `json:"open_trips"`
	OpenOrders        int64     `json:"open_orders"`
	LowStockParts     int64     `json:"low_stock_parts"`
	ExpiringDocuments int64     `json:"expiring_documents"`
	GeneratedAt       time.Time `json:"generated_at"`
}

// VehicleUtilization 车辆利用率报表项。
type VehicleUtilization struct {
	VehicleID       int64   `json:"vehicle_id"`
	PlateNumber     string  `json:"plate_number"`
	Model           string  `json:"model"`
	TripCount       int64   `json:"trip_count"`
	TotalDistance   int64   `json:"total_distance_km"`
	TotalFuelLiters int64   `json:"total_fuel_liters"`
	UtilizationPct  float64 `json:"utilization_pct"`
}

// FuelEfficiencyReport 油耗效率报表。
type FuelEfficiencyReport struct {
	VehicleID       int64   `json:"vehicle_id"`
	PlateNumber     string  `json:"plate_number"`
	TotalLiters     int64   `json:"total_liters_milli"`
	TotalDistanceKM int64   `json:"total_distance_km"`
	TotalCostCents  int64   `json:"total_cost_cents"`
	LitersPer100KM  float64 `json:"liters_per_100km"`
	CostPerKM       int64   `json:"cost_per_km_cents"`
	AbnormalRecords int64   `json:"abnormal_records"`
}

// MaintenanceCostReport 维保成本报表。
type MaintenanceCostReport struct {
	VehicleID      int64  `json:"vehicle_id"`
	PlateNumber    string `json:"plate_number"`
	OrderCount     int64  `json:"order_count"`
	PartsCostCents int64  `json:"parts_cost_cents"`
	LaborCostCents int64  `json:"labor_cost_cents"`
	TotalCostCents int64  `json:"total_cost_cents"`
	DowntimeHours  int64  `json:"downtime_hours"`
}

// ReportQuery 报表时间范围与分页查询。
type ReportQuery struct {
	From time.Time
	To   time.Time
	Page
}
