package service_test

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/zhangkui/go-fleet-maintenance/internal/domain/entity"
	mysqlrepo "github.com/zhangkui/go-fleet-maintenance/internal/repository/mysql"
	"github.com/zhangkui/go-fleet-maintenance/internal/service"
)

func TestBug006_PermissionAggregation(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectQuery("WHERE ur.user_id=\\?$").WithArgs(int64(1)).WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow("vehicle:create").AddRow("maintenance:approve").AddRow("audit:read").AddRow("user:manage"))
	codes, err := mysqlrepo.NewRoleRepository(db).UserPermissionCodes(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(codes) != 4 {
		t.Fatalf("codes=%v", codes)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}

	values := []entity.Permission{{Code: "vehicle:create"}, {Code: "maintenance:approve"}, {Code: "audit:read"}}
	got, err := service.NewRBACService(bug001Roles{}, &permissionStub{values: values}, service.NewAuditService(&bug001Audit{})).RolePermissions(context.Background(), 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(values) {
		t.Fatalf("permissions=%v", got)
	}
}
