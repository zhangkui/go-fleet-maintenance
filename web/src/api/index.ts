import client from '@/api/client'
import type {
  AuditLog,
  AuthTokens,
  Driver,
  DriverSchedule,
  DriverVehicleBinding,
  DriverViolation,
  FleetSummary,
  FuelEfficiencyReport,
  FuelRecord,
  ListParams,
  MaintenanceCostReport,
  MaintenanceOrder,
  MaintenanceOrderDetail,
  MaintenanceOrderPart,
  MaintenancePolicy,
  MeResponse,
  PageResult,
  Part,
  PartStockMovement,
  Permission,
  Reminder,
  ReminderScanResult,
  Role,
  Trip,
  TripHandover,
  User,
  Vehicle,
  VehicleLicense,
  VehicleStatusHistory,
  VehicleUtilization,
} from '@/types'

// 将 ListParams 序列化为查询字符串参数（跳过空值）。
function qs(params: ListParams = {}): Record<string, string | number | boolean> {
  const out: Record<string, string | number | boolean> = {}
  for (const [k, v] of Object.entries(params)) {
    if (v === undefined || v === '' || v === null) continue
    out[k] = v as string | number | boolean
  }
  return out
}

// ===== Auth =====
export const api = {
  register: (body: { username: string; password: string; email: string; full_name: string }) =>
    client.post<User>('/api/auth/register', body).then((r) => r.data),
  login: (username: string, password: string) =>
    client.post<AuthTokens>('/api/auth/login', { username, password }).then((r) => r.data),
  refresh: (refresh_token: string) =>
    client.post<AuthTokens>('/api/auth/refresh', { refresh_token }).then((r) => r.data),
  logout: (refresh_token: string) =>
    client.post('/api/auth/logout', { refresh_token }).then((r) => r.data),
  me: () => client.get<MeResponse>('/api/me').then((r) => r.data),
  changePassword: (old_password: string, new_password: string) =>
    client.post('/api/me/password', { old_password, new_password }).then((r) => r.data),

  // ===== Users =====
  listUsers: (p: ListParams = {}) =>
    client.get<PageResult<User>>('/api/users', { params: qs(p) }).then((r) => r.data),
  getUser: (id: number) =>
    client.get<User>(`/api/users/${id}`).then((r) => r.data),
  toggleUser: (id: number, enable: boolean) =>
    client.post(`/api/users/${id}/toggle`, { enable }).then((r) => r.data),
  resetPassword: (id: number, new_password: string) =>
    client.post(`/api/users/${id}/reset-password`, { new_password }).then((r) => r.data),
  listAuditLogs: (p: ListParams = {}) =>
    client.get<PageResult<AuditLog>>('/api/audit-logs', { params: qs(p) }).then((r) => r.data),

  // ===== RBAC =====
  listRoles: () => client.get<Role[]>('/api/roles').then((r) => r.data),
  createRole: (body: { name: string; code: string; description?: string }) =>
    client.post<Role>('/api/roles', body).then((r) => r.data),
  listPermissions: () => client.get<Permission[]>('/api/permissions').then((r) => r.data),
  rolePermissions: (id: number) =>
    client.get<Permission[]>(`/api/roles/${id}/permissions`).then((r) => r.data),
  userRoles: (id: number) =>
    client.get<Role[]>(`/api/users/${id}/roles`).then((r) => r.data),
  assignRole: (userId: number, role_id: number) =>
    client.post(`/api/users/${userId}/roles`, { role_id }).then((r) => r.data),
  revokeRole: (userId: number, role_id: number) =>
    client.delete(`/api/users/${userId}/roles/${role_id}`).then((r) => r.data),
  grantPermission: (roleId: number, permission_id: number) =>
    client.post(`/api/roles/${roleId}/permissions`, { permission_id }).then((r) => r.data),

  // ===== Vehicles =====
  createVehicle: (body: Partial<Vehicle>) =>
    client.post<Vehicle>('/api/vehicles', body).then((r) => r.data),
  listVehicles: (p: ListParams = {}) =>
    client.get<PageResult<Vehicle>>('/api/vehicles', { params: qs(p) }).then((r) => r.data),
  getVehicle: (id: number) =>
    client.get<Vehicle>(`/api/vehicles/${id}`).then((r) => r.data),
  changeVehicleStatus: (id: number, status: string, reason: string) =>
    client.post(`/api/vehicles/${id}/status`, { status, reason }).then((r) => r.data),
  updateVehicleMileage: (id: number, odometer_km: number) =>
    client.post(`/api/vehicles/${id}/mileage`, { odometer_km }).then((r) => r.data),
  vehicleStatusHistory: (id: number) =>
    client.get<VehicleStatusHistory[]>(`/api/vehicles/${id}/status-history`).then((r) => r.data),
  addVehicleLicense: (id: number, body: Partial<VehicleLicense>) =>
    client.post<VehicleLicense>(`/api/vehicles/${id}/licenses`, body).then((r) => r.data),
  listVehicleLicenses: (id: number) =>
    client.get<VehicleLicense[]>(`/api/vehicles/${id}/licenses`).then((r) => r.data),

  // ===== Drivers =====
  createDriver: (body: Partial<Driver>) =>
    client.post<Driver>('/api/drivers', body).then((r) => r.data),
  listDrivers: (p: ListParams = {}) =>
    client.get<PageResult<Driver>>('/api/drivers', { params: qs(p) }).then((r) => r.data),
  getDriver: (id: number) =>
    client.get<Driver>(`/api/drivers/${id}`).then((r) => r.data),
  updateDriverStatus: (id: number, status: string) =>
    client.post(`/api/drivers/${id}/status`, { status }).then((r) => r.data),
  createBinding: (body: Partial<DriverVehicleBinding>) =>
    client.post<DriverVehicleBinding>('/api/drivers/bindings', body).then((r) => r.data),
  listBindings: (vehicleId: number) =>
    client.get<DriverVehicleBinding[]>(`/api/vehicles/${vehicleId}/bindings`).then((r) => r.data),
  createSchedule: (body: Partial<DriverSchedule>) =>
    client.post<DriverSchedule>('/api/drivers/schedules', body).then((r) => r.data),
  listSchedules: (driverId: number, from?: string, to?: string) => {
    const params: Record<string, string> = {}
    if (from) params.from = from
    if (to) params.to = to
    return client
      .get<DriverSchedule[]>(`/api/drivers/${driverId}/schedules`, { params })
      .then((r) => r.data)
  },
  createViolation: (driverId: number, body: Partial<DriverViolation>) =>
    client.post<DriverViolation>(`/api/drivers/${driverId}/violations`, body).then((r) => r.data),
  listViolations: (driverId: number) =>
    client.get<DriverViolation[]>(`/api/drivers/${driverId}/violations`).then((r) => r.data),

  // ===== Trips =====
  createTrip: (body: Partial<Trip>) =>
    client.post<Trip>('/api/trips', body).then((r) => r.data),
  listTrips: (p: ListParams = {}) =>
    client.get<PageResult<Trip>>('/api/trips', { params: qs(p) }).then((r) => r.data),
  getTrip: (id: number) =>
    client.get<Trip>(`/api/trips/${id}`).then((r) => r.data),
  startTrip: (id: number) =>
    client.post(`/api/trips/${id}/start`).then((r) => r.data),
  completeTrip: (id: number, body: { end_odometer_km: number; completed_at: string; note: string }) =>
    client.post<Trip>(`/api/trips/${id}/complete`, body).then((r) => r.data),
  cancelTrip: (id: number) =>
    client.post(`/api/trips/${id}/cancel`).then((r) => r.data),
  createHandover: (tripId: number, body: Partial<TripHandover>) =>
    client.post<TripHandover>(`/api/trips/${tripId}/handovers`, body).then((r) => r.data),

  // ===== Fuel =====
  createFuelRecord: (body: {
    vehicle_id: number
    liters_milli: number
    unit_price_cents: number
    odometer_km: number
    total_cost_cents: number
    recorded_at: string
    idempotency_key: string
  }) => client.post<FuelRecord>('/api/fuel-records', body).then((r) => r.data),
  listFuelRecords: (p: ListParams = {}) =>
    client.get<PageResult<FuelRecord>>('/api/fuel-records', { params: qs(p) }).then((r) => r.data),
  getFuelRecord: (id: number) =>
    client.get<FuelRecord>(`/api/fuel-records/${id}`).then((r) => r.data),

  // ===== Maintenance =====
  createPolicy: (body: Partial<MaintenancePolicy>) =>
    client.post<MaintenancePolicy>('/api/maintenance/policies', body).then((r) => r.data),
  listPolicies: () =>
    client.get<MaintenancePolicy[]>('/api/maintenance/policies').then((r) => r.data),
  createOrder: (body: Partial<MaintenanceOrder> & { parts?: Partial<MaintenanceOrderPart>[] }) =>
    client.post<MaintenanceOrder>('/api/maintenance/orders', body).then((r) => r.data),
  listOrders: (p: ListParams = {}) =>
    client.get<PageResult<MaintenanceOrder>>('/api/maintenance/orders', { params: qs(p) }).then((r) => r.data),
  getOrder: (id: number) =>
    client.get<MaintenanceOrderDetail>(`/api/maintenance/orders/${id}`).then((r) => r.data),
  transitionOrder: (id: number, status: string) =>
    client.post(`/api/maintenance/orders/${id}/status`, { status }).then((r) => r.data),
  completeOrder: (id: number, body: { downtime_end: string; note: string }) =>
    client.post(`/api/maintenance/orders/${id}/complete`, body).then((r) => r.data),
  triggerDue: () =>
    client.post<{ created: number }>('/api/maintenance/trigger-due').then((r) => r.data),

  // ===== Parts =====
  createPart: (body: Partial<Part>) =>
    client.post<Part>('/api/parts', body).then((r) => r.data),
  listParts: (p: ListParams = {}) =>
    client.get<PageResult<Part>>('/api/parts', { params: qs(p) }).then((r) => r.data),
  listLowStock: () =>
    client.get<Part[]>('/api/parts/low-stock').then((r) => r.data),
  adjustStock: (id: number, body: { change: number; reason: string; note: string }) =>
    client.post<Part>(`/api/parts/${id}/adjust`, body).then((r) => r.data),
  listStockMovements: (id: number, p: ListParams = {}) =>
    client.get<PageResult<PartStockMovement>>(`/api/parts/${id}/movements`, { params: qs(p) }).then((r) => r.data),

  // ===== Reminders =====
  scanReminders: (days?: number) =>
    client
      .post<ReminderScanResult>('/api/reminders/scan', {}, { params: days ? { days } : {} })
      .then((r) => r.data),
  listReminders: (p: ListParams = {}) =>
    client.get<PageResult<Reminder>>('/api/reminders', { params: qs(p) }).then((r) => r.data),

  // ===== Reports =====
  fleetSummary: () =>
    client.get<FleetSummary>('/api/reports/fleet-summary').then((r) => r.data),
  vehicleUtilization: (p: ListParams = {}) =>
    client.get<PageResult<VehicleUtilization>>('/api/reports/vehicle-utilization', { params: qs(p) }).then((r) => r.data),
  fuelEfficiency: (p: ListParams = {}) =>
    client.get<PageResult<FuelEfficiencyReport>>('/api/reports/fuel-efficiency', { params: qs(p) }).then((r) => r.data),
  maintenanceCost: (p: ListParams = {}) =>
    client.get<PageResult<MaintenanceCostReport>>('/api/reports/maintenance-cost', { params: qs(p) }).then((r) => r.data),
}
