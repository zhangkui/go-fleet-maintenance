package service_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/zhangkui/go-fleet-maintenance/internal/domain/entity"
	"github.com/zhangkui/go-fleet-maintenance/internal/platform/redisx"
	"github.com/zhangkui/go-fleet-maintenance/internal/repository"
	mysqlrepo "github.com/zhangkui/go-fleet-maintenance/internal/repository/mysql"
	"github.com/zhangkui/go-fleet-maintenance/internal/service"
	"github.com/zhangkui/go-fleet-maintenance/internal/transport/http/handler"
)

type reportBehaviorStub struct {
	repository.ReportRepository
	summary  entity.FleetSummary
	from, to time.Time
}

func (s *reportBehaviorStub) FleetSummary(context.Context) (entity.FleetSummary, error) {
	return s.summary, nil
}
func (s *reportBehaviorStub) FuelEfficiency(_ context.Context, from, to time.Time, _ entity.Page) ([]entity.FuelEfficiencyReport, int64, error) {
	s.from, s.to = from, to
	return nil, 0, nil
}

type userBehaviorStub struct {
	repository.UserRepository
	user          entity.User
	filter        entity.Filter
	updatedID     int64
	updatedStatus string
}

func (s *userBehaviorStub) GetUserByID(context.Context, int64) (entity.User, error) {
	return s.user, nil
}
func (s *userBehaviorStub) UpdateUserStatus(_ context.Context, id int64, status string) error {
	s.updatedID, s.updatedStatus = id, status
	return nil
}
func (s *userBehaviorStub) ListUsers(_ context.Context, _ entity.Page, filter entity.Filter) ([]entity.User, int64, error) {
	s.filter = filter
	return nil, 0, nil
}

type auditListStub struct {
	repository.AuditRepository
	filter  entity.Filter
	entries []entity.AuditLog
}

func (s *auditListStub) Append(_ context.Context, entry entity.AuditLog) (entity.AuditLog, error) {
	s.entries = append(s.entries, entry)
	return entry, nil
}
func (s *auditListStub) List(_ context.Context, _ entity.Page, filter entity.Filter) ([]entity.AuditLog, int64, error) {
	s.filter = filter
	return nil, 0, nil
}

func TestBug025_FleetSummaryCacheTTL(t *testing.T) {
	server := miniredis.RunT(t)
	client, err := redisx.New(server.Addr(), "", 0, "bug025:")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := client.SetCache(ctx, "no-expiry", []byte("x"), 0); err != nil {
		t.Fatal(err)
	}
	if ttl := server.TTL("bug025:cache:no-expiry"); ttl != 0 {
		t.Fatalf("zero TTL should remain persistent, got %v", ttl)
	}
	repo := &reportBehaviorStub{summary: entity.FleetSummary{TotalVehicles: 3}}
	if _, err := service.NewReportService(repo, client).FleetSummary(ctx); err != nil {
		t.Fatal(err)
	}
	ttl := server.TTL("bug025:cache:fleet:summary")
	if ttl < 59*time.Second || ttl > 60*time.Second {
		t.Fatalf("summary ttl=%v", ttl)
	}
}

func TestBug026_FuelEfficiencyReport(t *testing.T) {
	repo := &reportBehaviorStub{}
	if _, _, err := service.NewReportService(repo, nil).FuelEfficiency(context.Background(), entity.ReportQuery{}); err != nil {
		t.Fatal(err)
	}
	if !repo.from.Before(repo.to) || repo.to.Sub(repo.from) < 27*24*time.Hour {
		t.Fatalf("default range from=%v to=%v", repo.from, repo.to)
	}
	db, mock, _ := sqlmock.New()
	defer db.Close()
	from, to := time.Now().AddDate(0, -1, 0), time.Now()
	mock.ExpectQuery("SELECT COUNT\\(DISTINCT vehicle_id\\)").WithArgs(from, to).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery("FROM vehicles v JOIN fuel_records").WithArgs(from, to, 20, 0).WillReturnRows(sqlmock.NewRows([]string{"id", "plate", "liters", "distance", "cost", "abnormal"}).AddRow(1, "A", 10000, 100, 5000, 0))
	rows, _, err := mysqlrepo.NewReportRepository(db).FuelEfficiency(context.Background(), from, to, entity.Page{Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].LitersPer100KM != 10 {
		t.Fatalf("report=%+v", rows)
	}
}

func TestBug027_AuditAppendFidelity(t *testing.T) {
	capture := &auditListStub{}
	service.NewAuditService(capture).Record(context.Background(), entity.AuditActor{UserID: 7, Username: "admin", IP: "10.0.0.7"}, "part.adjust", "part", 3, map[string]int64{"balance": 4})
	if len(capture.entries) != 1 || !strings.Contains(capture.entries[0].Detail, "balance") {
		t.Fatalf("entry=%+v", capture.entries)
	}
	db, mock, _ := sqlmock.New()
	defer db.Close()
	mock.ExpectExec("INSERT INTO audit_logs").WithArgs(int64(7), "admin", "part.adjust", "part", int64(3), `{"balance":4}`, "10.0.0.7").WillReturnResult(sqlmock.NewResult(1, 1))
	_, err := mysqlrepo.NewAuditRepository(db).Append(context.Background(), entity.AuditLog{ActorUserID: 7, ActorName: "admin", Action: "part.adjust", ResourceType: "part", ResourceID: 3, Detail: `{"balance":4}`, IP: "10.0.0.7"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestBug028_AuditActionFilter(t *testing.T) {
	audits := &auditListStub{}
	svc := service.NewUserService(&userBehaviorStub{}, service.NewAuditService(audits))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/api/audit-logs?action=auth.login", nil)
	handler.NewUserHandler(svc, 1024).AuditLogs(recorder, request)
	if audits.filter.Status != "auth.login" {
		t.Fatalf("filter=%q", audits.filter.Status)
	}
	db, mock, _ := sqlmock.New()
	defer db.Close()
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM audit_logs WHERE 1=1 AND action=\\?").WithArgs("auth.login").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("FROM audit_logs WHERE 1=1 AND action=\\?").WithArgs("auth.login", 20, 0).WillReturnRows(sqlmock.NewRows([]string{"id", "actor_user_id", "actor_name", "action", "resource_type", "resource_id", "detail", "ip", "created_at"}))
	if _, _, err := mysqlrepo.NewAuditRepository(db).List(context.Background(), entity.Page{Limit: 20}, entity.Filter{Status: "auth.login"}); err != nil {
		t.Fatal(err)
	}
}

func TestBug029_UserStatusAudit(t *testing.T) {
	users := &userBehaviorStub{user: entity.User{ID: 4, Status: entity.UserStatusActive}}
	audits := &auditListStub{}
	svc := service.NewUserService(users, service.NewAuditService(audits))
	if err := svc.ToggleUserStatus(context.Background(), 4, false, entity.AuditActor{UserID: 1}); err != nil {
		t.Fatal(err)
	}
	if users.updatedID != 4 || users.updatedStatus != entity.UserStatusDisabled {
		t.Fatalf("update id=%d status=%q", users.updatedID, users.updatedStatus)
	}
	if len(audits.entries) != 1 || audits.entries[0].Action != entity.AuditUserToggle {
		t.Fatalf("audit=%+v", audits.entries)
	}
	db, mock, _ := sqlmock.New()
	defer db.Close()
	mock.ExpectExec("UPDATE users SET status=\\? WHERE id=\\?").WithArgs(entity.UserStatusDisabled, int64(4)).WillReturnResult(sqlmock.NewResult(0, 1))
	if err := mysqlrepo.NewUserRepository(db).UpdateUserStatus(context.Background(), 4, entity.UserStatusDisabled); err != nil {
		t.Fatal(err)
	}
}

func TestBug030_UserStatusVisibility(t *testing.T) {
	users := &userBehaviorStub{}
	svc := service.NewUserService(users, service.NewAuditService(&auditListStub{}))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/api/users?limit=20", nil)
	handler.NewUserHandler(svc, 1024).List(recorder, request)
	if users.filter.Status != "" {
		t.Fatalf("default status=%q", users.filter.Status)
	}
	db, mock, _ := sqlmock.New()
	defer db.Close()
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM users WHERE 1=1").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("FROM users WHERE 1=1 ORDER BY id DESC LIMIT \\? OFFSET \\?").WithArgs(20, 0).WillReturnRows(sqlmock.NewRows([]string{"id", "username", "email", "full_name", "status", "last_login_at", "created_at", "updated_at"}))
	if _, _, err := mysqlrepo.NewUserRepository(db).ListUsers(context.Background(), entity.Page{Limit: 20}, entity.Filter{}); err != nil {
		t.Fatal(err)
	}
}
