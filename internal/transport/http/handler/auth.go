package handler

import (
	"net/http"

	"github.com/zhangkui/go-fleet-maintenance/internal/domain/entity"
	"github.com/zhangkui/go-fleet-maintenance/internal/service"
	"github.com/zhangkui/go-fleet-maintenance/internal/transport/http/middleware"
	"github.com/zhangkui/go-fleet-maintenance/internal/transport/http/request"
	"github.com/zhangkui/go-fleet-maintenance/internal/transport/http/response"
)

// AuthHandler 认证 HTTP 处理器。
type AuthHandler struct {
	svc      *service.AuthService
	maxBytes int64
}

// NewAuthHandler 构造认证处理器。
func NewAuthHandler(svc *service.AuthService, maxBytes int64) *AuthHandler {
	return &AuthHandler{svc: svc, maxBytes: maxBytes}
}

// Register POST /api/auth/register
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req request.RegisterRequest
	if err := request.Decode(r, h.maxBytes, &req); err != nil {
		response.Error(w, err)
		return
	}
	u, err := h.svc.Register(r.Context(), req.Username, req.Password, req.Email, req.FullName)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, u)
}

// Login POST /api/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req request.LoginRequest
	if err := request.Decode(r, h.maxBytes, &req); err != nil {
		response.Error(w, err)
		return
	}
	ip := clientIP(r)
	tokens, err := h.svc.Login(r.Context(), req.Username, req.Password, ip)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.OK(w, tokens)
}

// Refresh POST /api/auth/refresh
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req request.RefreshRequest
	if err := request.Decode(r, h.maxBytes, &req); err != nil {
		response.Error(w, err)
		return
	}
	ip := clientIP(r)
	tokens, err := h.svc.Refresh(r.Context(), req.RefreshToken, ip)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.OK(w, tokens)
}

// Logout POST /api/auth/logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req request.RefreshRequest
	if err := request.Decode(r, h.maxBytes, &req); err != nil {
		response.Error(w, err)
		return
	}
	actor := middleware.ActorFromContext(r.Context())
	_ = h.svc.Logout(r.Context(), req.RefreshToken, actor)
	response.NoContent(w)
}

// Me GET /api/me
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	actor := middleware.ActorFromContext(r.Context())
	u, perms, roles, err := h.svc.Me(r.Context(), actor.UserID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.OK(w, map[string]interface{}{
		"id":          u.ID,
		"username":    u.Username,
		"email":       u.Email,
		"full_name":   u.FullName,
		"status":      u.Status,
		"roles":       roles,
		"permissions": perms,
	})
}

// ChangePassword POST /api/me/password
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var req entity.PasswordChange
	if err := request.Decode(r, h.maxBytes, &req); err != nil {
		response.Error(w, err)
		return
	}
	actor := middleware.ActorFromContext(r.Context())
	if err := h.svc.ChangePassword(r.Context(), actor.UserID, req.OldPassword, req.NewPassword); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

// ResetPassword POST /api/users/{id}/reset-password （管理员）
func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	id := pathID(r)
	if id == 0 {
		response.Error(w, errInvalidID)
		return
	}
	var req struct {
		NewPassword string `json:"new_password"`
	}
	if err := request.Decode(r, h.maxBytes, &req); err != nil {
		response.Error(w, err)
		return
	}
	actor := middleware.ActorFromContext(r.Context())
	if err := h.svc.ResetPassword(r.Context(), id, req.NewPassword, actor); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}
