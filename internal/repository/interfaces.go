// Package repository 定义全部持久化接口，实现层通过接口隔离 MySQL 与 Redis。
package repository

import (
	"context"
	"errors"
	"time"

	"github.com/zhangkui/go-fleet-maintenance/internal/domain"
	"github.com/zhangkui/go-fleet-maintenance/internal/domain/entity"
)

// IsNotFound 报告错误是否为“资源不存在”。
func IsNotFound(err error) bool {
	return errors.Is(err, domain.ErrNotFound)
}

// UserRepository 用户持久化。
type UserRepository interface {
	CreateUser(ctx context.Context, u entity.User) (entity.User, error)
	GetUserByID(ctx context.Context, id int64) (entity.User, error)
	GetUserByUsername(ctx context.Context, username string) (entity.User, error)
	ListUsers(ctx context.Context, page entity.Page, filter entity.Filter) ([]entity.User, int64, error)
	UpdateUserStatus(ctx context.Context, id int64, status string) error
	UpdateUserPassword(ctx context.Context, id int64, hash string) error
	UpdateLastLogin(ctx context.Context, id int64, at time.Time) error
	IncFailedLogin(ctx context.Context, id int64) (int, error)
	ResetFailedLogin(ctx context.Context, id int64) error
	LockUser(ctx context.Context, id int64, until time.Time) error
}

// RoleRepository 角色持久化。
type RoleRepository interface {
	CreateRole(ctx context.Context, r entity.Role) (entity.Role, error)
	GetRoleByID(ctx context.Context, id int64) (entity.Role, error)
	GetRoleByCode(ctx context.Context, code string) (entity.Role, error)
	ListRoles(ctx context.Context) ([]entity.Role, error)
	AssignRole(ctx context.Context, userID, roleID int64) error
	RevokeRole(ctx context.Context, userID, roleID int64) error
	UserRoles(ctx context.Context, userID int64) ([]entity.Role, error)
	UserPermissionCodes(ctx context.Context, userID int64) ([]string, error)
}

// PermissionRepository 权限持久化。
type PermissionRepository interface {
	CreatePermission(ctx context.Context, p entity.Permission) (entity.Permission, error)
	ListPermissions(ctx context.Context) ([]entity.Permission, error)
	GrantToRole(ctx context.Context, roleID, permID int64) error
	RevokeFromRole(ctx context.Context, roleID, permID int64) error
	RolePermissions(ctx context.Context, roleID int64) ([]entity.Permission, error)
}

// SessionRepository 刷新令牌持久化（可撤销）。
type SessionRepository interface {
	SaveRefreshToken(ctx context.Context, t entity.RefreshToken) error
	GetRefreshToken(ctx context.Context, hash string) (entity.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, hash string, at time.Time) error
	RevokeAllForUser(ctx context.Context, userID int64, at time.Time) error
}

// AuditRepository 审计日志持久化。
type AuditRepository interface {
	Append(ctx context.Context, a entity.AuditLog) (entity.AuditLog, error)
	List(ctx context.Context, page entity.Page, filter entity.Filter) ([]entity.AuditLog, int64, error)
}

// VehicleRepository 车辆持久化。
type VehicleRepository interface {
	CreateVehicle(ctx context.Context, v entity.Vehicle) (entity.Vehicle, error)
	GetVehicleByID(ctx context.Context, id int64) (entity.Vehicle, error)
	GetVehicleByIDForUpdate(ctx context.Context, id int64) (entity.Vehicle, error)
	ListVehicles(ctx context.Context, page entity.Page, filter entity.Filter, sort entity.Sort) ([]entity.Vehicle, int64, error)
	UpdateVehicleStatus(ctx context.Context, id int64, status string) error
	UpdateVehicleMileage(ctx context.Context, id int64, odometer int64, at time.Time) error
	UpdateVehicleLicenseExpiry(ctx context.Context, id int64, kind string, expiry time.Time) error
	AppendVehicleStatusHistory(ctx context.Context, h entity.VehicleStatusHistory) error
	ListVehicleStatusHistory(ctx context.Context, vehicleID int64) ([]entity.VehicleStatusHistory, error)
	CreateVehicleLicense(ctx context.Context, l entity.VehicleLicense) (entity.VehicleLicense, error)
	ListVehicleLicenses(ctx context.Context, vehicleID int64) ([]entity.VehicleLicense, error)
	ListExpiringDocuments(ctx context.Context, from, to time.Time) ([]entity.Vehicle, error)
}

// DriverRepository 司机持久化。
type DriverRepository interface {
	CreateDriver(ctx context.Context, d entity.Driver) (entity.Driver, error)
	GetDriverByID(ctx context.Context, id int64) (entity.Driver, error)
	ListDrivers(ctx context.Context, page entity.Page, filter entity.Filter) ([]entity.Driver, int64, error)
	UpdateDriverStatus(ctx context.Context, id int64, status string) error
	CreateBinding(ctx context.Context, b entity.DriverVehicleBinding) (entity.DriverVehicleBinding, error)
	ListActiveBindings(ctx context.Context, vehicleID int64, at time.Time) ([]entity.DriverVehicleBinding, error)
	HasActiveBindingForVehicle(ctx context.Context, vehicleID int64, at time.Time) (bool, error)
	CreateSchedule(ctx context.Context, s entity.DriverSchedule) (entity.DriverSchedule, error)
	ListScheduleByDriver(ctx context.Context, driverID int64, from, to time.Time) ([]entity.DriverSchedule, error)
	CreateViolation(ctx context.Context, v entity.DriverViolation) (entity.DriverViolation, error)
	ListViolations(ctx context.Context, driverID int64) ([]entity.DriverViolation, error)
	ListExpiringLicenses(ctx context.Context, from, to time.Time) ([]entity.Driver, error)
}

// TripRepository 出车任务持久化。
type TripRepository interface {
	CreateTrip(ctx context.Context, t entity.Trip) (entity.Trip, error)
	GetTripByID(ctx context.Context, id int64) (entity.Trip, error)
	GetTripByIDForUpdate(ctx context.Context, id int64) (entity.Trip, error)
	ListTrips(ctx context.Context, page entity.Page, filter entity.Filter, sort entity.Sort) ([]entity.Trip, int64, error)
	CompleteTrip(ctx context.Context, id int64, endOdometer int64, completedBy int64, at time.Time) error
	CancelTrip(ctx context.Context, id int64, by int64, at time.Time) error
	StartTrip(ctx context.Context, id int64, by int64, at time.Time) error
	AppendTripStatusHistory(ctx context.Context, h entity.TripStatusHistory) error
	CreateHandover(ctx context.Context, h entity.TripHandover) (entity.TripHandover, error)
}

// FuelRepository 油耗流水持久化。
type FuelRepository interface {
	CreateFuelRecord(ctx context.Context, f entity.FuelRecord) (entity.FuelRecord, error)
	GetFuelRecord(ctx context.Context, id int64) (entity.FuelRecord, error)
	ListFuelRecords(ctx context.Context, page entity.Page, filter entity.Filter, sort entity.Sort) ([]entity.FuelRecord, int64, error)
	LastFuelRecord(ctx context.Context, vehicleID int64) (entity.FuelRecord, error)
	RecentFuelRecords(ctx context.Context, vehicleID int64, limit int) ([]entity.FuelRecord, error)
}

// MaintenanceRepository 维保计划与工单持久化。
type MaintenanceRepository interface {
	CreatePolicy(ctx context.Context, p entity.MaintenancePolicy) (entity.MaintenancePolicy, error)
	GetPolicyByID(ctx context.Context, id int64) (entity.MaintenancePolicy, error)
	ListPolicies(ctx context.Context, vehicleID int64) ([]entity.MaintenancePolicy, error)
	ListDuePolicies(ctx context.Context, now time.Time) ([]entity.MaintenancePolicy, error)
	UpdatePolicyLastService(ctx context.Context, id int64, km int64, at time.Time) error
	CreateOrder(ctx context.Context, o entity.MaintenanceOrder, parts []entity.MaintenanceOrderPart) (entity.MaintenanceOrder, error)
	GetOrderByID(ctx context.Context, id int64) (entity.MaintenanceOrder, error)
	GetOrderByIDForUpdate(ctx context.Context, id int64) (entity.MaintenanceOrder, error)
	ListOrders(ctx context.Context, page entity.Page, filter entity.Filter, sort entity.Sort) ([]entity.MaintenanceOrder, int64, error)
	UpdateOrderStatus(ctx context.Context, id int64, status string) error
	CompleteOrder(ctx context.Context, id int64, downtimeEnd time.Time) error
	HasOpenOrderForPolicy(ctx context.Context, policyID int64) (bool, error)
	AppendOrderStatusHistory(ctx context.Context, h entity.MaintenanceOrderStatusHistory) error
	ListOrderParts(ctx context.Context, orderID int64) ([]entity.MaintenanceOrderPart, error)
}

// PartRepository 配件与库存流水持久化。
type PartRepository interface {
	CreatePart(ctx context.Context, p entity.Part) (entity.Part, error)
	GetPartByIDForUpdate(ctx context.Context, id int64) (entity.Part, error)
	ListParts(ctx context.Context, page entity.Page, filter entity.Filter) ([]entity.Part, int64, error)
	UpdateStock(ctx context.Context, id int64, delta int64, balance int64) error
	AppendStockMovement(ctx context.Context, m entity.PartStockMovement) error
	ListStockMovements(ctx context.Context, partID int64, page entity.Page) ([]entity.PartStockMovement, int64, error)
	ListLowStock(ctx context.Context) ([]entity.Part, error)
}

// ReminderRepository 到期提醒持久化。
type ReminderRepository interface {
	CreateReminder(ctx context.Context, r entity.Reminder) (entity.Reminder, error)
	ListPendingReminders(ctx context.Context, before time.Time) ([]entity.Reminder, error)
	MarkSent(ctx context.Context, id int64, at time.Time) error
	ListReminders(ctx context.Context, page entity.Page, filter entity.Filter) ([]entity.Reminder, int64, error)
}

// ReportRepository 报表只读查询。
type ReportRepository interface {
	FleetSummary(ctx context.Context) (entity.FleetSummary, error)
	VehicleUtilization(ctx context.Context, from, to time.Time, page entity.Page) ([]entity.VehicleUtilization, int64, error)
	FuelEfficiency(ctx context.Context, from, to time.Time, page entity.Page) ([]entity.FuelEfficiencyReport, int64, error)
	MaintenanceCost(ctx context.Context, from, to time.Time, page entity.Page) ([]entity.MaintenanceCostReport, int64, error)
}

// Transactor 事务边界，保证跨实体多表写入一致性。
type Transactor interface {
	WithinTx(ctx context.Context, fn func(Stores) error) error
}

// Stores 事务内可用的全量仓储集合，绑定到当前事务。
type Stores struct {
	Users       UserRepository
	Roles       RoleRepository
	Permissions PermissionRepository
	Sessions    SessionRepository
	Audit       AuditRepository
	Vehicles    VehicleRepository
	Drivers     DriverRepository
	Trips       TripRepository
	Fuel        FuelRepository
	Maintenance MaintenanceRepository
	Parts       PartRepository
	Reminders   ReminderRepository
	Reports     ReportRepository
}
