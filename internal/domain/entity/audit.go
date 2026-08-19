package entity

import "time"

// AuditLog 审计日志实体，所有关键写操作与状态变更必须写入。
type AuditLog struct {
	ID           int64     `json:"id"`
	ActorUserID  int64     `json:"actor_user_id"`
	ActorName    string    `json:"actor_name"`
	Action       string    `json:"action"`
	ResourceType string    `json:"resource_type"`
	ResourceID   int64     `json:"resource_id"`
	Detail       string    `json:"detail"`
	IP           string    `json:"ip"`
	CreatedAt    time.Time `json:"created_at"`
}

// 审计动作枚举（部分）。
const (
	AuditLogin           = "auth.login"
	AuditLoginFailed     = "auth.login_failed"
	AuditLogout          = "auth.logout"
	AuditRefresh         = "auth.refresh"
	AuditPasswordChange  = "user.password_change"
	AuditPasswordReset   = "user.password_reset"
	AuditUserToggle      = "user.toggle"
	AuditRoleAssign      = "rbac.role_assign"
	AuditRoleRevoke      = "rbac.role_revoke"
	AuditPermissionGrant = "rbac.permission_grant"
	AuditVehicleStatus   = "vehicle.status_change"
	AuditTripComplete    = "trip.complete"
	AuditFuelRecord      = "fuel.create"
	AuditOrderStatus     = "maintenance.order_status"
	AuditPartStock       = "part.stock_adjust"
)
