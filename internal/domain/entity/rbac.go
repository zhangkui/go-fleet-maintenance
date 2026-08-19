package entity

import "time"

// Role 角色实体。
type Role struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Code        string    `json:"code"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// Permission 权限实体，按资源+动作组织。
type Permission struct {
	ID       int64  `json:"id"`
	Code     string `json:"code"`
	Name     string `json:"name"`
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

// RoleWithPermissions 角色及其权限码集合。
type RoleWithPermissions struct {
	Role        Role     `json:"role"`
	Permissions []string `json:"permissions"`
}

// UserWithRoles 用户及其角色码集合。
type UserWithRoles struct {
	User  User     `json:"user"`
	Roles []string `json:"roles"`
}

// 系统内置角色码与权限码。
const (
	RoleAdmin    = "admin"
	RoleOperator = "operator"
	RoleApprover = "approver"
	RoleAuditor  = "auditor"
)

// 角色固定中文说明，用于首次建库与前端展示。
var RoleDescriptions = map[string]string{
	RoleAdmin:    "业务管理员：全量业务与权限管理",
	RoleOperator: "日常操作人员：出车、油耗、维保录入",
	RoleApprover: "审批或复核人员：工单审批、任务复核",
	RoleAuditor:  "只读审计人员：只读全部业务与审计日志",
}
