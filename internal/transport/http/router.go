// Package httpx 组装路由与全部中间件。
package httpx

import (
	"net/http"

	"github.com/zhangkui/go-fleet-maintenance/internal/auth"
	"github.com/zhangkui/go-fleet-maintenance/internal/transport/http/handler"
	"github.com/zhangkui/go-fleet-maintenance/internal/transport/http/middleware"
)

// Router 依赖聚合，便于 main 组装。
type Router struct {
	mux      *http.ServeMux
	tokens   *auth.TokenService
	checker  *handler.PermissionCheckerImpl
	maxBytes int64
}

// New 构造路由器。
func New(tokens *auth.TokenService, checker *handler.PermissionCheckerImpl, maxBytes int64) *Router {
	r := &Router{
		mux:      http.NewServeMux(),
		tokens:   tokens,
		checker:  checker,
		maxBytes: maxBytes,
	}
	return r
}

// permit 构造“需指定权限”的中间件链。
func (r *Router) permit(code string) func(http.Handler) http.Handler {
	return middleware.RequirePermission(r.tokens, r.checker, code)
}

// authed 构造“仅认证”中间件。
func (r *Router) authed() func(http.Handler) http.Handler { return middleware.Authenticate(r.tokens) }

// Handler 返回根 Handler，包含全部中间件与路由。
func (r *Router) Handler(h *Handlers) http.Handler {
	mux := r.mux

	// 公开路由
	mux.HandleFunc("GET /health", h.Health.Health)
	mux.HandleFunc("POST /api/auth/register", h.Auth.Register)
	mux.HandleFunc("POST /api/auth/login", h.Auth.Login)
	mux.HandleFunc("POST /api/auth/refresh", h.Auth.Refresh)

	// 需认证
	mux.Handle("POST /api/auth/logout", r.authed()(http.HandlerFunc(h.Auth.Logout)))
	mux.Handle("GET /api/me", r.authed()(http.HandlerFunc(h.Auth.Me)))
	mux.Handle("POST /api/me/password", r.authed()(http.HandlerFunc(h.Auth.ChangePassword)))

	// 用户管理
	mux.Handle("GET /api/users", r.permit("user:read")(http.HandlerFunc(h.User.List)))
	mux.Handle("GET /api/users/{id}", r.permit("user:read")(http.HandlerFunc(h.User.Get)))
	mux.Handle("POST /api/users/{id}/toggle", r.permit("user:manage")(http.HandlerFunc(h.User.Toggle)))
	mux.Handle("POST /api/users/{id}/reset-password", r.permit("user:manage")(http.HandlerFunc(h.Auth.ResetPassword)))
	mux.Handle("GET /api/audit-logs", r.permit("audit:read")(http.HandlerFunc(h.User.AuditLogs)))

	// RBAC
	mux.Handle("GET /api/roles", r.permit("role:read")(http.HandlerFunc(h.RBAC.ListRoles)))
	mux.Handle("POST /api/roles", r.permit("role:manage")(http.HandlerFunc(h.RBAC.CreateRole)))
	mux.Handle("GET /api/permissions", r.permit("role:read")(http.HandlerFunc(h.RBAC.ListPermissions)))
	mux.Handle("GET /api/roles/{id}/permissions", r.permit("role:read")(http.HandlerFunc(h.RBAC.RolePermissions)))
	mux.Handle("GET /api/users/{id}/roles", r.permit("role:read")(http.HandlerFunc(h.RBAC.UserRoles)))
	mux.Handle("POST /api/users/{id}/roles", r.permit("role:manage")(http.HandlerFunc(h.RBAC.AssignRole)))
	mux.Handle("DELETE /api/users/{id}/roles/{role_id}", r.permit("role:manage")(http.HandlerFunc(h.RBAC.RevokeRole)))
	mux.Handle("POST /api/roles/{id}/permissions", r.permit("role:manage")(http.HandlerFunc(h.RBAC.GrantPermission)))

	// 车辆
	mux.Handle("POST /api/vehicles", r.permit("vehicle:create")(http.HandlerFunc(h.Vehicle.Create)))
	mux.Handle("GET /api/vehicles", r.permit("vehicle:read")(http.HandlerFunc(h.Vehicle.List)))
	mux.Handle("GET /api/vehicles/{id}", r.permit("vehicle:read")(http.HandlerFunc(h.Vehicle.Get)))
	mux.Handle("POST /api/vehicles/{id}/status", r.permit("vehicle:status")(http.HandlerFunc(h.Vehicle.ChangeStatus)))
	mux.Handle("POST /api/vehicles/{id}/mileage", r.permit("vehicle:update")(http.HandlerFunc(h.Vehicle.UpdateMileage)))
	mux.Handle("GET /api/vehicles/{id}/status-history", r.permit("vehicle:read")(http.HandlerFunc(h.Vehicle.StatusHistory)))
	mux.Handle("POST /api/vehicles/{id}/licenses", r.permit("vehicle:update")(http.HandlerFunc(h.Vehicle.AddLicense)))
	mux.Handle("GET /api/vehicles/{id}/licenses", r.permit("vehicle:read")(http.HandlerFunc(h.Vehicle.Licenses)))

	// 司机
	mux.Handle("POST /api/drivers", r.permit("driver:create")(http.HandlerFunc(h.Driver.Create)))
	mux.Handle("GET /api/drivers", r.permit("driver:read")(http.HandlerFunc(h.Driver.List)))
	mux.Handle("GET /api/drivers/{id}", r.permit("driver:read")(http.HandlerFunc(h.Driver.Get)))
	mux.Handle("POST /api/drivers/{id}/status", r.permit("driver:update")(http.HandlerFunc(h.Driver.UpdateStatus)))
	mux.Handle("POST /api/drivers/bindings", r.permit("driver:update")(http.HandlerFunc(h.Driver.CreateBinding)))
	mux.Handle("GET /api/vehicles/{id}/bindings", r.permit("driver:read")(http.HandlerFunc(h.Driver.ListBindings)))
	mux.Handle("POST /api/drivers/schedules", r.permit("driver:update")(http.HandlerFunc(h.Driver.CreateSchedule)))
	mux.Handle("GET /api/drivers/{id}/schedules", r.permit("driver:read")(http.HandlerFunc(h.Driver.ListSchedule)))
	mux.Handle("POST /api/drivers/{id}/violations", r.permit("driver:update")(http.HandlerFunc(h.Driver.CreateViolation)))
	mux.Handle("GET /api/drivers/{id}/violations", r.permit("driver:read")(http.HandlerFunc(h.Driver.ListViolations)))

	// 任务
	mux.Handle("POST /api/trips", r.permit("trip:create")(http.HandlerFunc(h.Trip.Create)))
	mux.Handle("GET /api/trips", r.permit("trip:read")(http.HandlerFunc(h.Trip.List)))
	mux.Handle("GET /api/trips/{id}", r.permit("trip:read")(http.HandlerFunc(h.Trip.Get)))
	mux.Handle("POST /api/trips/{id}/start", r.permit("trip:update")(http.HandlerFunc(h.Trip.Start)))
	mux.Handle("POST /api/trips/{id}/complete", r.permit("trip:update")(http.HandlerFunc(h.Trip.Complete)))
	mux.Handle("POST /api/trips/{id}/cancel", r.permit("trip:update")(http.HandlerFunc(h.Trip.Cancel)))
	mux.Handle("POST /api/trips/{id}/handovers", r.permit("trip:update")(http.HandlerFunc(h.Trip.CreateHandover)))

	// 油耗
	mux.Handle("POST /api/fuel-records", r.permit("fuel:create")(http.HandlerFunc(h.Fuel.Record)))
	mux.Handle("GET /api/fuel-records", r.permit("fuel:read")(http.HandlerFunc(h.Fuel.List)))
	mux.Handle("GET /api/fuel-records/{id}", r.permit("fuel:read")(http.HandlerFunc(h.Fuel.Get)))

	// 维保
	mux.Handle("POST /api/maintenance/policies", r.permit("maintenance:create")(http.HandlerFunc(h.Maintenance.CreatePolicy)))
	mux.Handle("GET /api/maintenance/policies", r.permit("maintenance:read")(http.HandlerFunc(h.Maintenance.ListPolicies)))
	mux.Handle("POST /api/maintenance/orders", r.permit("maintenance:create")(http.HandlerFunc(h.Maintenance.CreateOrder)))
	mux.Handle("GET /api/maintenance/orders", r.permit("maintenance:read")(http.HandlerFunc(h.Maintenance.ListOrders)))
	mux.Handle("GET /api/maintenance/orders/{id}", r.permit("maintenance:read")(http.HandlerFunc(h.Maintenance.GetOrder)))
	mux.Handle("POST /api/maintenance/orders/{id}/status", r.permit("maintenance:approve")(http.HandlerFunc(h.Maintenance.TransitionOrder)))
	mux.Handle("POST /api/maintenance/orders/{id}/complete", r.permit("maintenance:update")(http.HandlerFunc(h.Maintenance.CompleteOrder)))
	mux.Handle("POST /api/maintenance/trigger-due", r.permit("maintenance:trigger")(http.HandlerFunc(h.Maintenance.TriggerDue)))

	// 配件
	mux.Handle("POST /api/parts", r.permit("part:create")(http.HandlerFunc(h.Part.Create)))
	mux.Handle("GET /api/parts", r.permit("part:read")(http.HandlerFunc(h.Part.List)))
	mux.Handle("GET /api/parts/low-stock", r.permit("part:read")(http.HandlerFunc(h.Part.ListLowStock)))
	mux.Handle("POST /api/parts/{id}/adjust", r.permit("part:update")(http.HandlerFunc(h.Part.AdjustStock)))
	mux.Handle("GET /api/parts/{id}/movements", r.permit("part:read")(http.HandlerFunc(h.Part.ListStockMovements)))

	// 提醒
	mux.Handle("POST /api/reminders/scan", r.permit("reminder:scan")(http.HandlerFunc(h.Reminder.Scan)))
	mux.Handle("GET /api/reminders", r.permit("reminder:read")(http.HandlerFunc(h.Reminder.List)))

	// 报表
	mux.Handle("GET /api/reports/fleet-summary", r.permit("report:read")(http.HandlerFunc(h.Report.FleetSummary)))
	mux.Handle("GET /api/reports/vehicle-utilization", r.permit("report:read")(http.HandlerFunc(h.Report.VehicleUtilization)))
	mux.Handle("GET /api/reports/fuel-efficiency", r.permit("report:read")(http.HandlerFunc(h.Report.FuelEfficiency)))
	mux.Handle("GET /api/reports/maintenance-cost", r.permit("report:read")(http.HandlerFunc(h.Report.MaintenanceCost)))

	// 全局中间件：恢复、请求体限制。
	return middleware.Recover(middleware.LimitBody(r.maxBytes)(mux))
}

// Handlers 聚合全部处理器，由 main 注入。
type Handlers struct {
	Auth        *handler.AuthHandler
	User        *handler.UserHandler
	RBAC        *handler.RBACHandler
	Vehicle     *handler.VehicleHandler
	Driver      *handler.DriverHandler
	Trip        *handler.TripHandler
	Fuel        *handler.FuelHandler
	Maintenance *handler.MaintenanceHandler
	Part        *handler.PartHandler
	Reminder    *handler.ReminderHandler
	Report      *handler.ReportHandler
	Health      *handler.HealthHandler
}
