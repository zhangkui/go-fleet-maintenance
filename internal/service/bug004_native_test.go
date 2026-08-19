package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/zhangkui/go-fleet-maintenance/internal/auth"
	"github.com/zhangkui/go-fleet-maintenance/internal/domain/entity"
	mysqlrepo "github.com/zhangkui/go-fleet-maintenance/internal/repository/mysql"
	"github.com/zhangkui/go-fleet-maintenance/internal/service"
)

func TestBug004_RegistrationAssignsOperatorRole(t *testing.T) {
	t.Run("repository finds role by code", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		rows := sqlmock.NewRows([]string{"id", "name", "code", "description", "created_at"}).AddRow(3, "Operator", entity.RoleOperator, "daily operator", time.Now())
		mock.ExpectQuery("SELECT id,name,code,description,created_at FROM roles WHERE code=\\?").WithArgs(entity.RoleOperator).WillReturnRows(rows)
		role, err := mysqlrepo.NewRoleRepository(db).GetRoleByCode(context.Background(), entity.RoleOperator)
		if err != nil {
			t.Fatal(err)
		}
		if role.Code != entity.RoleOperator {
			t.Fatalf("role=%q", role.Code)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})

	roles := &bug004Roles{}
	svc := service.NewAuthService(&bug001Users{}, roles, bug001Sessions{}, service.NewAuditService(&bug001Audit{}), nil, auth.NewTokenService("secret", "test", time.Hour), time.Hour, 5, time.Minute)
	user, err := svc.Register(context.Background(), "new_operator", "Password123!", "operator@example.com", "Operator")
	if err != nil {
		t.Fatal(err)
	}
	if roles.requestedCode != entity.RoleOperator {
		t.Fatalf("requested role=%q", roles.requestedCode)
	}
	if roles.assignedUser != user.ID || roles.assignedRole != 3 {
		t.Fatalf("assignment user=%d role=%d", roles.assignedUser, roles.assignedRole)
	}
}

type bug004Roles struct {
	requestedCode              string
	assignedUser, assignedRole int64
}

func (*bug004Roles) CreateRole(context.Context, entity.Role) (entity.Role, error) {
	return entity.Role{}, nil
}
func (*bug004Roles) GetRoleByID(context.Context, int64) (entity.Role, error) {
	return entity.Role{}, nil
}
func (r *bug004Roles) GetRoleByCode(_ context.Context, code string) (entity.Role, error) {
	r.requestedCode = code
	if code == entity.RoleOperator {
		return entity.Role{ID: 3, Code: code}, nil
	}
	return entity.Role{}, nil
}
func (*bug004Roles) ListRoles(context.Context) ([]entity.Role, error) { return nil, nil }
func (r *bug004Roles) AssignRole(_ context.Context, userID, roleID int64) error {
	r.assignedUser = userID
	r.assignedRole = roleID
	return nil
}
func (*bug004Roles) RevokeRole(context.Context, int64, int64) error               { return nil }
func (*bug004Roles) UserRoles(context.Context, int64) ([]entity.Role, error)      { return nil, nil }
func (*bug004Roles) UserPermissionCodes(context.Context, int64) ([]string, error) { return nil, nil }
