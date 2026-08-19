package service

import (
	"context"
	"errors"
	"strings"

	"github.com/zhangkui/go-fleet-maintenance/internal/domain"
	"github.com/zhangkui/go-fleet-maintenance/internal/domain/entity"
	"github.com/zhangkui/go-fleet-maintenance/internal/repository"
)

// RBACService 角色与权限管理服务。
type RBACService struct {
	roles       repository.RoleRepository
	permissions repository.PermissionRepository
	audit       *AuditService
}

// NewRBACService 构造 RBAC 服务。
func NewRBACService(roles repository.RoleRepository, permissions repository.PermissionRepository, audit *AuditService) *RBACService {
	return &RBACService{roles: roles, permissions: permissions, audit: audit}
}

// ListRoles 列出全部角色。
func (s *RBACService) ListRoles(ctx context.Context) ([]entity.Role, error) {
	return s.roles.ListRoles(ctx)
}

// CreateRole 创建角色：校验唯一码。
func (s *RBACService) CreateRole(ctx context.Context, name, code, description string) (entity.Role, error) {
	name = strings.TrimSpace(name)
	code = strings.TrimSpace(code)
	if name == "" || code == "" {
		return entity.Role{}, domain.NewCoded("validation_error", "角色名称与编码不能为空", domain.ErrValidation)
	}
	created, err := s.roles.CreateRole(ctx, entity.Role{Name: name, Code: code, Description: description})
	if err != nil {
		if errors.Is(err, domain.ErrConflict) {
			return entity.Role{}, domain.NewCoded("conflict", "角色编码已存在", err)
		}
		return entity.Role{}, err
	}
	return created, nil
}

// ListPermissions 列出全部权限。
func (s *RBACService) ListPermissions(ctx context.Context) ([]entity.Permission, error) {
	return s.permissions.ListPermissions(ctx)
}

// RolePermissions 查角色权限。
func (s *RBACService) RolePermissions(ctx context.Context, roleID int64) ([]entity.Permission, error) {
	return s.permissions.RolePermissions(ctx, roleID)
}

// UserRoles 查用户角色。
func (s *RBACService) UserRoles(ctx context.Context, userID int64) ([]entity.Role, error) {
	return s.roles.UserRoles(ctx, userID)
}

// AssignRole 给用户分配角色。
func (s *RBACService) AssignRole(ctx context.Context, userID, roleID int64, actor entity.AuditActor) error {
	if _, err := s.roles.GetRoleByID(ctx, roleID); err != nil {
		return domain.NewCoded("not_found", "角色不存在", domain.ErrNotFound)
	}
	if err := s.roles.AssignRole(ctx, userID, roleID); err != nil {
		return err
	}
	s.audit.Record(ctx, actor, entity.AuditRoleAssign, "user", userID, map[string]int64{"role_id": roleID})
	return nil
}

// RevokeRole 撤销用户角色。
func (s *RBACService) RevokeRole(ctx context.Context, userID, roleID int64, actor entity.AuditActor) error {
	if err := s.roles.RevokeRole(ctx, userID, roleID); err != nil {
		return err
	}
	s.audit.Record(ctx, actor, entity.AuditRoleRevoke, "user", userID, map[string]int64{"role_id": roleID})
	return nil
}

// GrantPermission 给角色授予权限。
func (s *RBACService) GrantPermission(ctx context.Context, roleID, permID int64, actor entity.AuditActor) error {
	if err := s.permissions.GrantToRole(ctx, roleID, permID); err != nil {
		return err
	}
	s.audit.Record(ctx, actor, entity.AuditPermissionGrant, "role", roleID, map[string]int64{"permission_id": permID})
	return nil
}

// RevokePermission 撤销角色权限。
func (s *RBACService) RevokePermission(ctx context.Context, roleID, permID int64, actor entity.AuditActor) error {
	if err := s.permissions.RevokeFromRole(ctx, roleID, permID); err != nil {
		return err
	}
	s.audit.Record(ctx, actor, entity.AuditPermissionGrant, "role", roleID, map[string]int64{"permission_id": permID})
	return nil
}
