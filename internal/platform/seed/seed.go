// Package seed 负责首次启动时创建默认管理员、内置角色与权限。
package seed

import (
	"context"
	"log/slog"
	"time"

	"github.com/zhangkui/go-fleet-maintenance/internal/auth"
	"github.com/zhangkui/go-fleet-maintenance/internal/domain/entity"
	"github.com/zhangkui/go-fleet-maintenance/internal/repository"
)

// DefaultAdmin 创建本地默认管理员 admin / Admin123!（可由环境变量覆盖）。
func DefaultAdmin(ctx context.Context, users repository.UserRepository, roles repository.RoleRepository, perms repository.PermissionRepository, adminUser, adminPass string) error {
	_, err := users.GetUserByUsername(ctx, adminUser)
	if err == nil {
		return nil // 已存在
	}
	if !repository.IsNotFound(err) {
		return err
	}
	hash, err := auth.HashPassword(adminPass)
	if err != nil {
		return err
	}
	u, err := users.CreateUser(ctx, entity.User{
		Username: adminUser, PasswordHash: hash, Email: "admin@local", FullName: "本地管理员",
	})
	if err != nil {
		return err
	}
	// 确保内置角色存在。
	adminRole, _ := ensureRole(ctx, roles, "业务管理员", entity.RoleAdmin, entity.RoleDescriptions[entity.RoleAdmin])
	_, _ = ensureRole(ctx, roles, "日常操作人员", entity.RoleOperator, entity.RoleDescriptions[entity.RoleOperator])
	_, _ = ensureRole(ctx, roles, "审批或复核人员", entity.RoleApprover, entity.RoleDescriptions[entity.RoleApprover])
	_, _ = ensureRole(ctx, roles, "只读审计人员", entity.RoleAuditor, entity.RoleDescriptions[entity.RoleAuditor])
	// 内置权限。
	for _, code := range PermissionCodes {
		if _, err := perms.CreatePermission(ctx, entity.Permission{
			Code: code, Name: code, Resource: resourceOf(code), Action: actionOf(code),
		}); err != nil {
			// 已存在则忽略。
		}
	}
	// 给 admin 角色授予全部权限。
	allPerms, _ := perms.ListPermissions(ctx)
	for _, p := range allPerms {
		_ = perms.GrantToRole(ctx, adminRole.ID, p.ID)
	}
	// admin 角色分配给默认管理员。
	if err := roles.AssignRole(ctx, u.ID, adminRole.ID); err != nil {
		return err
	}
	// operator / approver / auditor 授权限。
	opRole, opErr := roles.GetRoleByCode(ctx, entity.RoleOperator)
	apRole, apErr := roles.GetRoleByCode(ctx, entity.RoleApprover)
	auRole, auErr := roles.GetRoleByCode(ctx, entity.RoleAuditor)
	for _, p := range allPerms {
		switch resourceOf(p.Code) {
		case "vehicle", "driver", "trip", "fuel", "maintenance", "part", "reminder", "report", "audit":
			act := actionOf(p.Code)
			if opErr == nil && (act == "read" || act == "create" || act == "update") {
				_ = perms.GrantToRole(ctx, opRole.ID, p.ID)
			}
			if apErr == nil && (act == "approve" || act == "update") {
				_ = perms.GrantToRole(ctx, apRole.ID, p.ID)
			}
			if auErr == nil && act == "read" {
				_ = perms.GrantToRole(ctx, auRole.ID, p.ID)
			}
		}
	}
	slog.Info("已创建默认管理员", "username", adminUser, "at", time.Now())
	return nil
}

func ensureRole(ctx context.Context, roles repository.RoleRepository, name, code, desc string) (entity.Role, error) {
	r, err := roles.GetRoleByCode(ctx, code)
	if err == nil {
		return r, nil
	}
	if !repository.IsNotFound(err) {
		return entity.Role{}, err
	}
	return roles.CreateRole(ctx, entity.Role{Name: name, Code: code, Description: desc})
}

// PermissionCodes 内置权限码集合。
var PermissionCodes = []string{
	"vehicle:create", "vehicle:read", "vehicle:update", "vehicle:status",
	"driver:create", "driver:read", "driver:update",
	"trip:create", "trip:read", "trip:update", "trip:approve",
	"fuel:create", "fuel:read",
	"maintenance:create", "maintenance:read", "maintenance:update", "maintenance:approve", "maintenance:trigger",
	"part:create", "part:read", "part:update",
	"reminder:read", "reminder:scan",
	"report:read",
	"audit:read",
	"user:read", "user:manage",
	"role:read", "role:manage",
}

func resourceOf(code string) string {
	for i := 0; i < len(code); i++ {
		if code[i] == ':' {
			return code[:i]
		}
	}
	return code
}

func actionOf(code string) string {
	for i := 0; i < len(code); i++ {
		if code[i] == ':' {
			return code[i+1:]
		}
	}
	return code
}
