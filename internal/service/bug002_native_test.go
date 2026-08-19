package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/zhangkui/go-fleet-maintenance/internal/auth"
	"github.com/zhangkui/go-fleet-maintenance/internal/domain"
	"github.com/zhangkui/go-fleet-maintenance/internal/domain/entity"
	mysqlrepo "github.com/zhangkui/go-fleet-maintenance/internal/repository/mysql"
	"github.com/zhangkui/go-fleet-maintenance/internal/service"
)

func TestBug002_RefreshTokenRotation(t *testing.T) {
	t.Run("repository revokes by token hash", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		mock.ExpectExec("UPDATE refresh_tokens SET revoked_at=\\? WHERE token_hash=\\? AND revoked_at IS NULL").WithArgs(sqlmock.AnyArg(), "old-hash").WillReturnResult(sqlmock.NewResult(0, 1))
		if err := mysqlrepo.NewSessionRepository(db).RevokeRefreshToken(context.Background(), "old-hash", time.Now()); err != nil {
			t.Fatal(err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})

	oldToken := "refresh-token-before-rotation"
	oldHash := auth.HashToken(oldToken)
	sessions := &bug002Sessions{tokens: map[string]entity.RefreshToken{oldHash: {UserID: 1, TokenHash: oldHash, ExpiresAt: time.Now().Add(time.Hour)}}}
	users := &bug001Users{user: entity.User{ID: 1, Username: "admin", Status: entity.UserStatusActive}}
	svc := service.NewAuthService(users, bug001Roles{}, sessions, service.NewAuditService(&bug001Audit{}), nil, auth.NewTokenService("secret", "test", time.Hour), time.Hour, 5, time.Minute)

	rotated, err := svc.Refresh(context.Background(), oldToken, "127.0.0.1")
	if err != nil {
		t.Fatalf("first refresh: %v", err)
	}
	if sessions.revoked != oldHash {
		t.Fatalf("revoked=%q, want hash %q", sessions.revoked, oldHash)
	}
	if rotated.RefreshToken == "" || rotated.RefreshToken == oldToken {
		t.Fatal("new refresh token was not rotated")
	}
	if _, err := svc.Refresh(context.Background(), oldToken, "127.0.0.1"); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("old token reuse err=%v", err)
	}
	if _, err := svc.Refresh(context.Background(), rotated.RefreshToken, "127.0.0.1"); err != nil {
		t.Fatalf("new token rejected: %v", err)
	}
}

type bug002Sessions struct {
	tokens  map[string]entity.RefreshToken
	revoked string
}

func (s *bug002Sessions) SaveRefreshToken(_ context.Context, token entity.RefreshToken) error {
	s.tokens[token.TokenHash] = token
	return nil
}
func (s *bug002Sessions) GetRefreshToken(_ context.Context, hash string) (entity.RefreshToken, error) {
	token, ok := s.tokens[hash]
	if !ok {
		return entity.RefreshToken{}, domain.ErrNotFound
	}
	return token, nil
}
func (s *bug002Sessions) RevokeRefreshToken(_ context.Context, hash string, at time.Time) error {
	s.revoked = hash
	token, ok := s.tokens[hash]
	if ok {
		token.RevokedAt = &at
		s.tokens[hash] = token
	}
	return nil
}
func (s *bug002Sessions) RevokeAllForUser(context.Context, int64, time.Time) error { return nil }
