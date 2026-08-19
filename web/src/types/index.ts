// 领域类型定义，字段名与 Go entity json tag 完全一致。

export interface User {
  id: number
  username: string
  email: string
  full_name: string
  status: string
  last_login_at?: string
  created_at: string
  updated_at: string
}

export interface AuthTokens {
  access_token: string
  refresh_token: string
  token_type: string
  expires_in: number
  expires_at: string
  user: User
  permissions: string[]
}

export interface MeResponse {
  id: number
  username: string
  email: string
  full_name: string
  status: string
  roles: string[]
  permissions: string[]
}

export interface Role {
  id: number
  name: string
  code: string
  description: string
  created_at: string
}

export interface Permission {
  id: number
  code: string
  name: string
  resource: string
  action: string
}

export interface Vehicle {
  id: number
  model: string
  vin: string
  plate_number: string
  status: string
  odometer_km: number
  color: string
  engine_no: string
  purchase_date?: string
  insurance_expiry?: string
  inspection_expiry?: string
  created_at: string
  updated_at: string
}

export interface VehicleStatusHistory {
  id: number
  vehicle_id: number
  from_status: string
  to_status: string
  reason: string
  changed_by: number
  changed_at: string
}

export interface VehicleLicense {
  id: number
  vehicle_id: number
  kind: string
  number: string
  issue_date?: string
  expiry_date: string
  created_at: string
  updated_at: string
}

export interface Driver {
  id: number
  name: string
  license_number: string
  license_class: string
  license_expiry?: string
  phone: string
  status: string
  created_at: string
  updated_at: string
}

export interface DriverVehicleBinding {
  id: number
  driver_id: number
  vehicle_id: number
  start_date: string
  end_date?: string
  status: string
  created_at: string
}

export interface DriverSchedule {
  id: number
  driver_id: number
  shift_date: string
  shift_type: string
  note: string
  created_at: string
}

export interface DriverViolation {
  id: number
  driver_id: number
  occurred_at: string
  violation_type: string
  points: number
  fine_cents: number
  description: string
  status: string
  created_at: string
}

export interface Trip {
  id: number
  vehicle_id: number
  driver_id: number
  route: string
  load_kg: number
  start_odometer_km: number
  end_odometer_km?: number
  status: string
  completed_at?: string
  completed_by?: number
  note: string
  created_at: string
  updated_at: string
}

export interface TripHandover {
  id: number
  trip_id: number
  from_driver_id: number
  to_driver_id: number
  handover_at: string
  location: string
  note: string
  created_at: string
}

export interface FuelRecord {
  id: number
  vehicle_id: number
  liters_milli: number
  unit_price_cents: number
  odometer_km: number
  total_cost_cents: number
  abnormal: boolean
  recorded_at: string
  created_by: number
  created_at: string
}

export interface MaintenancePolicy {
  id: number
  vehicle_id: number
  name: string
  kind: string
  interval_km: number
  interval_days: number
  last_service_km: number
  last_service_at: string
  next_due_km: number
  next_due_at: string
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface MaintenanceOrder {
  id: number
  vehicle_id: number
  policy_id?: number
  kind: string
  title: string
  status: string
  downtime_start?: string
  downtime_end?: string
  parts_cost_cents: number
  labor_cost_cents: number
  total_cost_cents: number
  completed_at?: string
  created_by: number
  created_at: string
  updated_at: string
}

export interface MaintenanceOrderPart {
  id: number
  order_id: number
  part_id: number
  quantity: number
  unit_cost_cents: number
  line_total_cents: number
}

export interface MaintenanceOrderDetail {
  order: MaintenanceOrder
  parts: MaintenanceOrderPart[]
}

export interface Part {
  id: number
  sku: string
  name: string
  unit: string
  stock_quantity: number
  reorder_point: number
  unit_cost_cents: number
  created_at: string
  updated_at: string
}

export interface PartStockMovement {
  id: number
  part_id: number
  change_quantity: number
  reason: string
  ref_order_id?: number
  balance_after: number
  created_at: string
  created_by: number
}

export interface Reminder {
  id: number
  entity_type: string
  entity_id: number
  due_at: string
  message: string
  status: string
  created_at: string
  sent_at?: string
}

export interface ReminderScanResult {
  insurance: number
  inspection: number
  license: number
  maintenance: number
  created: number
}

export interface FleetSummary {
  total_vehicles: number
  active_vehicles: number
  in_maintenance: number
  total_drivers: number
  active_drivers: number
  open_trips: number
  open_orders: number
  low_stock_parts: number
  expiring_documents: number
  generated_at: string
}

export interface VehicleUtilization {
  vehicle_id: number
  plate_number: string
  model: string
  trip_count: number
  total_distance_km: number
  total_fuel_liters: number
  utilization_pct: number
}

export interface FuelEfficiencyReport {
  vehicle_id: number
  plate_number: string
  total_liters_milli: number
  total_distance_km: number
  total_cost_cents: number
  liters_per_100km: number
  cost_per_km_cents: number
  abnormal_records: number
}

export interface MaintenanceCostReport {
  vehicle_id: number
  plate_number: string
  order_count: number
  parts_cost_cents: number
  labor_cost_cents: number
  total_cost_cents: number
  downtime_hours: number
}

export interface AuditLog {
  id: number
  actor_user_id: number
  actor_name: string
  action: string
  resource_type: string
  resource_id: number
  detail: string
  ip: string
  created_at: string
}

// 通用分页结果包装。
export interface PageResult<T> {
  items: T[]
  total: number
  limit: number
  page: number
}

// 通用错误响应。
export interface ApiError {
  code: string
  message: string
  details?: Record<string, unknown>
}

// 列表查询通用参数。
export interface ListParams {
  limit?: number
  offset?: number
  status?: string
  keyword?: string
  vehicle_id?: number
  driver_id?: number
  kind?: string
  abnormal?: boolean
  from?: string
  to?: string
  sort?: string
  order?: string
}
