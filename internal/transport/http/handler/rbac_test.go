package handler

import (
	"context"
	"testing"

	"github.com/zhangkui/go-fleet-maintenance/internal/domain/entity"
)

// fakeRoleRepo 最小实现，用于 PermissionChecker 单测。
type fakeRoleRepo struct {
	perms map[int64][]string
}

func (r *fakeRoleRepo) UserPermissionCodes(_ context.Context, userID int64) ([]string, error) {
	return r.perms[userID], nil
}
func (r *fakeRoleRepo) CreateRole(_ context.Context, role entity.Role) (entity.Role, error) {
	return role, nil
}
func (r *fakeRoleRepo) GetRoleByID(_ context.Context, id int64) (entity.Role, error) {
	return entity.Role{}, nil
}
func (r *fakeRoleRepo) GetRoleByCode(_ context.Context, code string) (entity.Role, error) {
	return entity.Role{Code: code}, nil
}
func (r *fakeRoleRepo) ListRoles(_ context.Context) ([]entity.Role, error) {
	return nil, nil
}
func (r *fakeRoleRepo) AssignRole(_ context.Context, userID, roleID int64) error { return nil }
func (r *fakeRoleRepo) RevokeRole(_ context.Context, userID, roleID int64) error { return nil }
func (r *fakeRoleRepo) UserRoles(_ context.Context, userID int64) ([]entity.Role, error) {
	return nil, nil
}

func TestPermissionChecker_HasPermission(t *testing.T) {
	roles := &fakeRoleRepo{perms: map[int64][]string{7: {"vehicle:read", "fuel:create"}}}
	c := NewPermissionChecker(roles)
	ok, err := c.HasPermission(context.Background(), 7, "vehicle:read")
	if err != nil || !ok {
		t.Fatalf("期望拥有 vehicle:read 权限")
	}
	ok, err = c.HasPermission(context.Background(), 7, "vehicle:delete")
	if err != nil || ok {
		t.Fatalf("不期望拥有 vehicle:delete 权限")
	}
}
