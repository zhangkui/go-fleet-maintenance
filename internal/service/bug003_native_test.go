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

func TestBug003_PasswordChangeRevokesUserSessions(t *testing.T) {
	t.Run("repository targets active sessions", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		mock.ExpectExec("UPDATE refresh_tokens SET revoked_at=\\? WHERE user_id=\\? AND revoked_at IS NULL").WithArgs(sqlmock.AnyArg(), int64(9)).WillReturnResult(sqlmock.NewResult(0, 2))
		if err := mysqlrepo.NewSessionRepository(db).RevokeAllForUser(context.Background(), 9, time.Now()); err != nil {
			t.Fatal(err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})

	hash, err := auth.HashPassword("OldPassword123!")
	if err != nil {
		t.Fatal(err)
	}
	users := &bug001Users{user: entity.User{ID: 9, Username: "operator", PasswordHash: hash, Status: entity.UserStatusActive}}
	sessions := &bug003Sessions{}
	svc := service.NewAuthService(users, bug001Roles{}, sessions, service.NewAuditService(&bug001Audit{}), nil, auth.NewTokenService("secret", "test", time.Hour), time.Hour, 5, time.Minute)
	if err := svc.ChangePassword(context.Background(), 9, "OldPassword123!", "NewPassword123!"); err != nil {
		t.Fatal(err)
	}
	if sessions.userID != 9 {
		t.Fatalf("revoked user=%d, want 9", sessions.userID)
	}
}

type bug003Sessions struct{ userID int64 }

func (*bug003Sessions) SaveRefreshToken(context.Context, entity.RefreshToken) error { return nil }
func (*bug003Sessions) GetRefreshToken(context.Context, string) (entity.RefreshToken, error) {
	return entity.RefreshToken{}, nil
}
func (*bug003Sessions) RevokeRefreshToken(context.Context, string, time.Time) error { return nil }
func (s *bug003Sessions) RevokeAllForUser(_ context.Context, userID int64, _ time.Time) error {
	s.userID = userID
	return nil
}
