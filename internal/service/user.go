package service

import (
	"context"

	"github.com/zhangkui/go-fleet-maintenance/internal/domain/entity"
	"github.com/zhangkui/go-fleet-maintenance/internal/repository"
)

// UserService 用户管理服务：启停、查询。
type UserService struct {
	users repository.UserRepository
	audit *AuditService
}

// NewUserService 构造用户管理服务。
func NewUserService(users repository.UserRepository, audit *AuditService) *UserService {
	return &UserService{users: users, audit: audit}
}

// ListUsers 分页查询用户。
func (s *UserService) ListUsers(ctx context.Context, page entity.Page, filter entity.Filter) ([]entity.User, int64, error) {
	return s.users.ListUsers(ctx, page, filter)
}

// GetUser 查用户详情。
func (s *UserService) GetUser(ctx context.Context, id int64) (entity.User, error) {
	return s.users.GetUserByID(ctx, id)
}

// ToggleUserStatus 启停用户。
func (s *UserService) ToggleUserStatus(ctx context.Context, id int64, enable bool, actor entity.AuditActor) error {
	u, err := s.users.GetUserByID(ctx, id)
	if err != nil {
		return err
	}
	status := entity.UserStatusActive
	if !enable {
		status = entity.UserStatusDisabled
	}
	if u.Status == status {
		return nil
	}
	// 状态未更新成功时不记录成功审计；更新成功后再补记 user.toggle 审计。
	if err := s.users.UpdateUserStatus(ctx, id, status); err != nil {
		return err
	}
	s.audit.Record(ctx, actor, entity.AuditUserToggle, "user", id, map[string]string{"status": status})
	return nil
}

// ListAuditLogs 分页查询审计日志。
func (s *UserService) ListAuditLogs(ctx context.Context, page entity.Page, filter entity.Filter) ([]entity.AuditLog, int64, error) {
	return s.audit.repo.List(ctx, page, filter)
}
