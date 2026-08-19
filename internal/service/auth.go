package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/zhangkui/go-fleet-maintenance/internal/auth"
	"github.com/zhangkui/go-fleet-maintenance/internal/domain"
	"github.com/zhangkui/go-fleet-maintenance/internal/domain/entity"
	"github.com/zhangkui/go-fleet-maintenance/internal/platform/redisx"
	"github.com/zhangkui/go-fleet-maintenance/internal/repository"
)

// AuthService 认证服务：注册、登录、刷新、退出、修改密码。
type AuthService struct {
	users      repository.UserRepository
	roles      repository.RoleRepository
	sessions   repository.SessionRepository
	audit      *AuditService
	redis      *redisx.Client
	tokens     *auth.TokenService
	refreshTTL time.Duration
	rateLimit  int
	rateWindow time.Duration
}

// NewAuthService 构造认证服务。
func NewAuthService(
	users repository.UserRepository,
	roles repository.RoleRepository,
	sessions repository.SessionRepository,
	audit *AuditService,
	redis *redisx.Client,
	tokens *auth.TokenService,
	refreshTTL time.Duration,
	rateLimit int,
	rateWindow time.Duration,
) *AuthService {
	return &AuthService{
		users: users, roles: roles, sessions: sessions, audit: audit, redis: redis,
		tokens: tokens, refreshTTL: refreshTTL, rateLimit: rateLimit, rateWindow: rateWindow,
	}
}

// Register 注册新用户，默认 operator 角色。
func (s *AuthService) Register(ctx context.Context, username, password, email, fullName string) (entity.User, error) {
	username = strings.TrimSpace(username)
	if !validUsername(username) {
		return entity.User{}, domain.NewCoded("validation_error", "用户名只能包含字母数字下划线，长度 3-64", domain.ErrValidation)
	}
	if len(password) < 8 {
		return entity.User{}, domain.NewCoded("validation_error", "密码长度不能少于 8 位", domain.ErrValidation)
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return entity.User{}, domain.NewCoded("validation_error", err.Error(), domain.ErrValidation)
	}
	u, err := s.users.CreateUser(ctx, entity.User{Username: username, PasswordHash: hash, Email: email, FullName: fullName})
	if err != nil {
		if errors.Is(err, domain.ErrConflict) {
			return entity.User{}, domain.NewCoded("conflict", "用户名已存在", err)
		}
		return entity.User{}, err
	}
	// 默认分配 operator 角色：通过角色持久化按 code 查到标准 operator 角色并完成关联。
	if role, err := s.roles.GetRoleByCode(ctx, entity.RoleOperator); err == nil {
		_ = s.roles.AssignRole(ctx, u.ID, role.ID)
	}
	s.audit.Record(ctx, entity.AuditActor{UserID: u.ID, Username: u.Username}, entity.AuditLogin, "user", u.ID, nil)
	return u, nil
}

// Login 登录：限流、锁定、bcrypt 校验、签发令牌对、审计。
func (s *AuthService) Login(ctx context.Context, username, password, ip string) (entity.AuthTokens, error) {
	// 登录失败限流：按用户名+IP 维度计数。
	rlKey := "login:" + username + ":" + ip
	if _, over, _ := s.redis.HitRateLimit(ctx, rlKey, s.rateWindow, s.rateLimit); over {
		s.audit.Record(ctx, entity.AuditActor{Username: username, IP: ip}, entity.AuditLoginFailed, "user", 0, map[string]string{"reason": "rate_limited"})
		return entity.AuthTokens{}, domain.ErrRateLimited
	}
	u, err := s.users.GetUserByUsername(ctx, username)
	if err != nil {
		s.audit.Record(ctx, entity.AuditActor{Username: username, IP: ip}, entity.AuditLoginFailed, "user", 0, map[string]string{"reason": "user_not_found"})
		return entity.AuthTokens{}, domain.ErrUnauthorized
	}
	if u.Status != entity.UserStatusActive {
		s.audit.Record(ctx, entity.AuditActor{UserID: u.ID, Username: u.Username, IP: ip}, entity.AuditLoginFailed, "user", u.ID, map[string]string{"reason": "disabled"})
		return entity.AuthTokens{}, domain.ErrUnauthorized
	}
	if u.LockedUntil != nil && u.LockedUntil.After(time.Now()) {
		return entity.AuthTokens{}, domain.ErrRateLimited
	}
	if !auth.VerifyPassword(u.PasswordHash, password) {
		count, _ := s.users.IncFailedLogin(ctx, u.ID)
		if count > s.rateLimit+1 {
			_ = s.users.LockUser(ctx, u.ID, time.Now().Add(s.rateWindow))
		}
		s.audit.Record(ctx, entity.AuditActor{UserID: u.ID, Username: u.Username, IP: ip}, entity.AuditLoginFailed, "user", u.ID, map[string]int{"failed_count": count})
		return entity.AuthTokens{}, domain.ErrUnauthorized
	}
	perms, _ := s.roles.UserPermissionCodes(ctx, u.ID)
	roles, _ := s.roles.UserRoles(ctx, u.ID)
	roleCodes := make([]string, 0, len(roles))
	for _, r := range roles {
		roleCodes = append(roleCodes, r.Code)
	}
	// 签发访问令牌与刷新令牌。
	access, err := s.tokens.Sign(auth.Claims{UserID: u.ID, Username: u.Username, Roles: roleCodes})
	if err != nil {
		return entity.AuthTokens{}, err
	}
	refresh, err := auth.RandomToken()
	if err != nil {
		return entity.AuthTokens{}, err
	}
	_ = s.sessions.SaveRefreshToken(ctx, entity.RefreshToken{
		UserID:    u.ID,
		TokenHash: auth.HashToken(refresh),
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(s.refreshTTL),
		ClientIP:  ip,
	})
	_ = s.users.UpdateLastLogin(ctx, u.ID, time.Now())
	s.audit.Record(ctx, entity.AuditActor{UserID: u.ID, Username: u.Username, IP: ip}, entity.AuditLogin, "user", u.ID, nil)
	return entity.AuthTokens{
		AccessToken:  access,
		RefreshToken: refresh,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.tokensTTL().Seconds()),
		ExpiresAt:    time.Now().Add(s.tokensTTL()),
		User:         u,
		Permissions:  perms,
	}, nil
}

func (s *AuthService) tokensTTL() time.Duration { return 30 * time.Minute }

// Refresh 刷新会话：校验刷新令牌有效性，签发新令牌对。
func (s *AuthService) Refresh(ctx context.Context, refreshToken, ip string) (entity.AuthTokens, error) {
	hash := auth.HashToken(refreshToken)
	t, err := s.sessions.GetRefreshToken(ctx, hash)
	if err != nil {
		return entity.AuthTokens{}, domain.ErrUnauthorized
	}
	if t.RevokedAt != nil || t.ExpiresAt.Before(time.Now()) {
		return entity.AuthTokens{}, domain.ErrUnauthorized
	}
	u, err := s.users.GetUserByID(ctx, t.UserID)
	if err != nil {
		return entity.AuthTokens{}, domain.ErrUnauthorized
	}
	if u.Status != entity.UserStatusActive {
		return entity.AuthTokens{}, domain.ErrUnauthorized
	}
	// 撤销旧刷新令牌，签发新的（刷新令牌轮转）。
	_ = s.sessions.RevokeRefreshToken(ctx, refreshToken, time.Now())
	roles, _ := s.roles.UserRoles(ctx, u.ID)
	roleCodes := make([]string, 0, len(roles))
	for _, r := range roles {
		roleCodes = append(roleCodes, r.Code)
	}
	perms, _ := s.roles.UserPermissionCodes(ctx, u.ID)
	access, err := s.tokens.Sign(auth.Claims{UserID: u.ID, Username: u.Username, Roles: roleCodes})
	if err != nil {
		return entity.AuthTokens{}, err
	}
	newRefresh, err := auth.RandomToken()
	if err != nil {
		return entity.AuthTokens{}, err
	}
	_ = s.sessions.SaveRefreshToken(ctx, entity.RefreshToken{
		UserID:    u.ID,
		TokenHash: auth.HashToken(newRefresh),
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(s.refreshTTL),
		ClientIP:  ip,
	})
	s.audit.Record(ctx, entity.AuditActor{UserID: u.ID, Username: u.Username, IP: ip}, entity.AuditRefresh, "user", u.ID, nil)
	return entity.AuthTokens{
		AccessToken:  access,
		RefreshToken: newRefresh,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.tokensTTL().Seconds()),
		ExpiresAt:    time.Now().Add(s.tokensTTL()),
		User:         u,
		Permissions:  perms,
	}, nil
}

// Logout 撤销刷新令牌。
func (s *AuthService) Logout(ctx context.Context, refreshToken string, actor entity.AuditActor) error {
	hash := auth.HashToken(refreshToken)
	_ = s.sessions.RevokeRefreshToken(ctx, hash, time.Now())
	s.audit.Record(ctx, actor, entity.AuditLogout, "user", actor.UserID, nil)
	return nil
}

// ChangePassword 修改密码：校验旧密码、更新、撤销全部刷新令牌。
func (s *AuthService) ChangePassword(ctx context.Context, userID int64, oldPassword, newPassword string) error {
	u, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if !auth.VerifyPassword(u.PasswordHash, oldPassword) {
		return domain.NewCoded("validation_error", "旧密码不正确", domain.ErrValidation)
	}
	if len(newPassword) < 8 {
		return domain.NewCoded("validation_error", "新密码长度不能少于 8 位", domain.ErrValidation)
	}
	hash, err := auth.HashPassword(newPassword)
	if err != nil {
		return domain.NewCoded("validation_error", err.Error(), domain.ErrValidation)
	}
	if err := s.users.UpdateUserPassword(ctx, userID, hash); err != nil {
		return err
	}
	_ = s.sessions.RevokeAllForUser(ctx, 0, time.Now())
	s.audit.Record(ctx, entity.AuditActor{UserID: userID, Username: u.Username}, entity.AuditPasswordChange, "user", userID, nil)
	return nil
}

// ResetPassword 管理员重置密码。
func (s *AuthService) ResetPassword(ctx context.Context, targetUserID int64, newPassword string, actor entity.AuditActor) error {
	if len(newPassword) < 8 {
		return domain.NewCoded("validation_error", "新密码长度不能少于 8 位", domain.ErrValidation)
	}
	u, err := s.users.GetUserByID(ctx, targetUserID)
	if err != nil {
		return err
	}
	hash, err := auth.HashPassword(newPassword)
	if err != nil {
		return domain.NewCoded("validation_error", err.Error(), domain.ErrValidation)
	}
	if err := s.users.UpdateUserPassword(ctx, targetUserID, hash); err != nil {
		return err
	}
	_ = s.sessions.RevokeAllForUser(ctx, targetUserID, time.Now())
	s.audit.Record(ctx, actor, entity.AuditPasswordReset, "user", targetUserID, map[string]string{"target": u.Username})
	return nil
}

func validUsername(s string) bool {
	if len(s) < 3 || len(s) > 64 {
		return false
	}
	for _, c := range s {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_') {
			return false
		}
	}
	return true
}

// Me 返回当前用户及其角色与权限。
func (s *AuthService) Me(ctx context.Context, userID int64) (entity.User, []string, []string, error) {
	u, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return entity.User{}, nil, nil, err
	}
	perms, _ := s.roles.UserPermissionCodes(ctx, userID)
	roles, _ := s.roles.UserRoles(ctx, userID)
	codes := make([]string, 0, len(roles))
	for _, r := range roles {
		codes = append(codes, r.Code)
	}
	return u, perms, codes, nil
}
