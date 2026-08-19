package handler

import (
	"net/http"

	"github.com/zhangkui/go-fleet-maintenance/internal/service"
	"github.com/zhangkui/go-fleet-maintenance/internal/transport/http/middleware"
	"github.com/zhangkui/go-fleet-maintenance/internal/transport/http/request"
	"github.com/zhangkui/go-fleet-maintenance/internal/transport/http/response"
)

// UserHandler 用户管理 HTTP 处理器。
type UserHandler struct {
	svc      *service.UserService
	maxBytes int64
}

// NewUserHandler 构造用户管理处理器。
func NewUserHandler(svc *service.UserService, maxBytes int64) *UserHandler {
	return &UserHandler{svc: svc, maxBytes: maxBytes}
}

// List GET /api/users
func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	page := request.Page(r)
	filter := request.Filter(r)
	if filter.Status == "" {
		filter.Status = "active"
	}
	users, total, err := h.svc.ListUsers(r.Context(), page, filter)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Page(w, users, total, page.Limit, page.Offset/page.Limit+1)
}

// Get GET /api/users/{id}
func (h *UserHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := pathID(r)
	if id == 0 {
		response.Error(w, errInvalidID)
		return
	}
	u, err := h.svc.GetUser(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.OK(w, u)
}

// Toggle POST /api/users/{id}/toggle
func (h *UserHandler) Toggle(w http.ResponseWriter, r *http.Request) {
	id := pathID(r)
	if id == 0 {
		response.Error(w, errInvalidID)
		return
	}
	var req struct {
		Enable bool `json:"enable"`
	}
	if err := request.Decode(r, h.maxBytes, &req); err != nil {
		response.Error(w, err)
		return
	}
	actor := middleware.ActorFromContext(r.Context())
	if err := h.svc.ToggleUserStatus(r.Context(), id, req.Enable, actor); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

// AuditLogs GET /api/audit-logs
func (h *UserHandler) AuditLogs(w http.ResponseWriter, r *http.Request) {
	page := request.Page(r)
	filter := request.Filter(r)
	filter.Status = r.URL.Query().Get("action")
	logs, total, err := h.svc.ListAuditLogs(r.Context(), page, filter)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Page(w, logs, total, page.Limit, page.Offset/page.Limit+1)
}
