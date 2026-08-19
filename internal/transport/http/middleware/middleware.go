// Package middleware 提供 HTTP 中间件：认证、鉴权、限流、请求大小限制、恢复。
package middleware

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/zhangkui/go-fleet-maintenance/internal/auth"
	"github.com/zhangkui/go-fleet-maintenance/internal/domain"
	"github.com/zhangkui/go-fleet-maintenance/internal/domain/entity"
)

type ctxKey int

const (
	ctxActorKey ctxKey = iota
)

// WithActor 把操作发起者写入上下文。
func WithActor(ctx context.Context, actor entity.AuditActor) context.Context {
	return context.WithValue(ctx, ctxActorKey, actor)
}

// ActorFromContext 取出操作发起者，未认证时返回零值。
func ActorFromContext(ctx context.Context) entity.AuditActor {
	a, _ := ctx.Value(ctxActorKey).(entity.AuditActor)
	return a
}

// Authenticate 解析访问令牌并注入操作者上下文。
func Authenticate(tokens *auth.TokenService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if len(header) < 8 || header[:7] != "Bearer " {
				writeAuthError(w, domain.ErrUnauthorized)
				return
			}
			claims, err := tokens.Verify(header[7:])
			if err != nil {
				writeAuthError(w, domain.ErrUnauthorized)
				return
			}
			actor := entity.AuditActor{UserID: claims.UserID, Username: claims.Username, Role: firstRole(claims.Roles), IP: clientIP(r)}
			ctx := WithActor(r.Context(), actor)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequirePermission 要求请求具备指定权限码，否则 403。
func RequirePermission(tokens *auth.TokenService, checker PermissionChecker, code string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if len(header) < 8 || header[:7] != "Bearer " {
				writeAuthError(w, domain.ErrUnauthorized)
				return
			}
			claims, err := tokens.Verify(header[7:])
			if err != nil {
				writeAuthError(w, domain.ErrUnauthorized)
				return
			}
			// admin 角色直接放行。
			if hasRole(claims.Roles, entity.RoleAdmin) {
				actor := entity.AuditActor{UserID: claims.UserID, Username: claims.Username, Role: entity.RoleAdmin, IP: clientIP(r)}
				next.ServeHTTP(w, r.WithContext(WithActor(r.Context(), actor)))
				return
			}
			allowed, err := checker.HasPermission(r.Context(), claims.UserID, code)
			if err != nil || !allowed {
				writeAuthError(w, domain.ErrForbidden)
				return
			}
			roles := claims.Roles
			actor := entity.AuditActor{UserID: claims.UserID, Username: claims.Username, Role: firstRole(roles), IP: clientIP(r)}
			next.ServeHTTP(w, r.WithContext(WithActor(r.Context(), actor)))
		})
	}
}

// PermissionChecker 权限校验接口，由 handler 层注入实现。
type PermissionChecker interface {
	HasPermission(ctx context.Context, userID int64, code string) (bool, error)
}

// LimitBody 限制请求体大小。
func LimitBody(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}

// Recover 捕获 panic，返回 500 并记录栈。
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic", "rec", rec, "stack", string(debug.Stack()))
				writeJSON(w, http.StatusInternalServerError, map[string]string{"code": "internal_error", "message": "服务器内部错误"})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// JSON 请求头设置。
func JSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

func writeAuthError(w http.ResponseWriter, err error) {
	code, status := "unauthorized", http.StatusUnauthorized
	if err == domain.ErrForbidden {
		code, status = "forbidden", http.StatusForbidden
	}
	if err == domain.ErrRateLimited {
		code, status = "rate_limited", http.StatusTooManyRequests
	}
	writeJSON(w, status, map[string]string{"code": code, "message": err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func clientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	return r.RemoteAddr
}

func firstRole(roles []string) string {
	if len(roles) > 0 {
		return roles[0]
	}
	return ""
}

func hasRole(roles []string, code string) bool {
	for _, r := range roles {
		if r == code {
			return true
		}
	}
	return false
}
