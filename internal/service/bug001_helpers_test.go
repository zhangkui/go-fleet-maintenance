package service_test

import (
	"context"
	"time"

	"github.com/zhangkui/go-fleet-maintenance/internal/domain/entity"
)

type bug001Users struct {
	user   entity.User
	failed int
	locked bool
}

func (r *bug001Users) CreateUser(_ context.Context, user entity.User) (entity.User, error) {
	user.ID = 11
	return user, nil
}
func (r *bug001Users) GetUserByID(context.Context, int64) (entity.User, error) { return r.user, nil }
func (r *bug001Users) GetUserByUsername(context.Context, string) (entity.User, error) {
	u := r.user
	if r.locked {
		until := time.Now().Add(time.Minute)
		u.LockedUntil = &until
	}
	return u, nil
}
func (r *bug001Users) ListUsers(context.Context, entity.Page, entity.Filter) ([]entity.User, int64, error) {
	return nil, 0, nil
}
func (r *bug001Users) UpdateUserStatus(context.Context, int64, string) error   { return nil }
func (r *bug001Users) UpdateUserPassword(context.Context, int64, string) error { return nil }
func (r *bug001Users) UpdateLastLogin(context.Context, int64, time.Time) error {
	r.failed = 0
	r.locked = false
	return nil
}
func (r *bug001Users) IncFailedLogin(context.Context, int64) (int, error) {
	r.failed++
	return r.failed, nil
}
func (r *bug001Users) ResetFailedLogin(context.Context, int64) error {
	r.failed = 0
	r.locked = false
	return nil
}
func (r *bug001Users) LockUser(context.Context, int64, time.Time) error { r.locked = true; return nil }

type bug001Roles struct{}

func (bug001Roles) CreateRole(context.Context, entity.Role) (entity.Role, error) {
	return entity.Role{}, nil
}
func (bug001Roles) GetRoleByID(context.Context, int64) (entity.Role, error) {
	return entity.Role{}, nil
}
func (bug001Roles) GetRoleByCode(context.Context, string) (entity.Role, error) {
	return entity.Role{}, nil
}
func (bug001Roles) ListRoles(context.Context) ([]entity.Role, error)             { return nil, nil }
func (bug001Roles) AssignRole(context.Context, int64, int64) error               { return nil }
func (bug001Roles) RevokeRole(context.Context, int64, int64) error               { return nil }
func (bug001Roles) UserRoles(context.Context, int64) ([]entity.Role, error)      { return nil, nil }
func (bug001Roles) UserPermissionCodes(context.Context, int64) ([]string, error) { return nil, nil }

type bug001Sessions struct{}

func (bug001Sessions) SaveRefreshToken(context.Context, entity.RefreshToken) error { return nil }
func (bug001Sessions) GetRefreshToken(context.Context, string) (entity.RefreshToken, error) {
	return entity.RefreshToken{}, nil
}
func (bug001Sessions) RevokeRefreshToken(context.Context, string, time.Time) error { return nil }
func (bug001Sessions) RevokeAllForUser(context.Context, int64, time.Time) error    { return nil }

type bug001Audit struct{}

func (*bug001Audit) Append(_ context.Context, log entity.AuditLog) (entity.AuditLog, error) {
	return log, nil
}
func (*bug001Audit) List(context.Context, entity.Page, entity.Filter) ([]entity.AuditLog, int64, error) {
	return nil, 0, nil
}
