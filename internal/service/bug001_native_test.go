package service_test

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/zhangkui/go-fleet-maintenance/internal/auth"
	"github.com/zhangkui/go-fleet-maintenance/internal/domain"
	"github.com/zhangkui/go-fleet-maintenance/internal/domain/entity"
	"github.com/zhangkui/go-fleet-maintenance/internal/platform/redisx"
	mysqlrepo "github.com/zhangkui/go-fleet-maintenance/internal/repository/mysql"
	"github.com/zhangkui/go-fleet-maintenance/internal/service"
)

func TestBug001_AuthenticationLockout(t *testing.T) {
	t.Run("rate limit", func(t *testing.T) {
		server := miniredis.RunT(t)
		client, err := redisx.New(server.Addr(), "", 0, "test:")
		if err != nil {
			t.Fatal(err)
		}
		for attempt := 1; attempt <= 6; attempt++ {
			count, over, err := client.HitRateLimit(context.Background(), "login:admin:127.0.0.1", time.Minute, 5)
			if err != nil {
				t.Fatal(err)
			}
			if count != attempt {
				t.Fatalf("attempt %d: count=%d", attempt, count)
			}
			if over != (attempt > 5) {
				t.Fatalf("attempt %d: over=%v", attempt, over)
			}
		}
	})

	t.Run("incremented repository count", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		mock.ExpectExec("UPDATE users SET failed_login_count=failed_login_count\\+1 WHERE id=\\?").WithArgs(int64(7)).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectQuery("SELECT failed_login_count FROM users WHERE id=\\?").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"failed_login_count"}).AddRow(5))
		count, err := mysqlrepo.NewUserRepository(db).IncFailedLogin(context.Background(), 7)
		if err != nil {
			t.Fatal(err)
		}
		if count != 5 {
			t.Fatalf("count=%d, want 5", count)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})

	hash, err := auth.HashPassword("Admin123!")
	if err != nil {
		t.Fatal(err)
	}
	users := &bug001Users{user: entity.User{ID: 1, Username: "admin", PasswordHash: hash, Status: entity.UserStatusActive}}
	server := miniredis.RunT(t)
	client, err := redisx.New(server.Addr(), "", 0, "test:")
	if err != nil {
		t.Fatal(err)
	}
	svc := service.NewAuthService(users, bug001Roles{}, bug001Sessions{}, service.NewAuditService(&bug001Audit{}), client, auth.NewTokenService("secret", "test", time.Hour), time.Hour, 5, time.Minute)

	t.Run("correct password", func(t *testing.T) {
		if _, err := svc.Login(context.Background(), "admin", "Admin123!", "success-ip"); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("wrong password", func(t *testing.T) {
		users.failed, users.locked = 0, false
		_, err := svc.Login(context.Background(), "admin", "WrongPwd123!", "wrong-ip")
		if !errors.Is(err, domain.ErrUnauthorized) {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("lock threshold", func(t *testing.T) {
		users.failed, users.locked = 0, false
		for attempt := 1; attempt <= 5; attempt++ {
			_, err := svc.Login(context.Background(), "admin", "WrongPwd123!", "threshold-ip-"+strconv.Itoa(attempt))
			if !errors.Is(err, domain.ErrUnauthorized) {
				t.Fatalf("attempt %d: %v", attempt, err)
			}
		}
		if !users.locked {
			t.Fatal("account was not locked at five failures")
		}
		if users.failed != 5 {
			t.Fatalf("failed count=%d", users.failed)
		}
	})
	t.Run("successful reset", func(t *testing.T) {
		users.failed, users.locked = 3, false
		if _, err := svc.Login(context.Background(), "admin", "Admin123!", "reset-ip"); err != nil {
			t.Fatal(err)
		}
		if users.failed != 0 {
			t.Fatalf("failed count=%d", users.failed)
		}
	})
}
