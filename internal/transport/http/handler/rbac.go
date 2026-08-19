package handler

import (
	"net/http"
	"strconv"

	"github.com/zhangkui/go-fleet-maintenance/internal/service"
	"github.com/zhangkui/go-fleet-maintenance/internal/transport/http/request"
	"github.com/zhangkui/go-fleet-maintenance/internal/transport/http/response"
)

// RBACHandler 角色与权限 HTTP 处理器。
type RBACHandler struct {
	svc      *service.RBACService
	maxBytes int64
}

// NewRBACHandler 构造 RBAC 处理器。
func NewRBACHandler(svc *service.RBACService, maxBytes int64) *RBACHandler {
	return &RBACHandler{svc: svc, maxBytes: maxBytes}
}

// ListRoles GET /api/roles
func (h *RBACHandler) ListRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := h.svc.ListRoles(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.OK(w, roles)
}

// CreateRole POST /api/roles
func (h *RBACHandler) CreateRole(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Code        string `json:"code"`
		Description string `json:"description"`
	}
	if err := request.Decode(r, h.maxBytes, &req); err != nil {
		response.Error(w, err)
		return
	}
	created, err := h.svc.CreateRole(r.Context(), req.Name, req.Code, req.Description)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, created)
}

// ListPermissions GET /api/permissions
func (h *RBACHandler) ListPermissions(w http.ResponseWriter, r *http.Request) {
	perms, err := h.svc.ListPermissions(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.OK(w, perms)
}

// RolePermissions GET /api/roles/{id}/permissions
func (h *RBACHandler) RolePermissions(w http.ResponseWriter, r *http.Request) {
	roleID, _ := strconv.ParseInt(r.URL.Query().Get("role_id"), 10, 64)
	if roleID == 0 {
		roleID = pathID(r)
	}
	perms, err := h.svc.RolePermissions(r.Context(), roleID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.OK(w, perms)
}

// UserRoles GET /api/users/{id}/roles
func (h *RBACHandler) UserRoles(w http.ResponseWriter, r *http.Request) {
	userID := pathID(r)
	roles, err := h.svc.UserRoles(r.Context(), userID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.OK(w, roles)
}

// AssignRole POST /api/users/{id}/roles
func (h *RBACHandler) AssignRole(w http.ResponseWriter, r *http.Request) {
	userID := pathID(r)
	var req struct {
		RoleID int64 `json:"role_id"`
	}
	if err := request.Decode(r, h.maxBytes, &req); err != nil {
		response.Error(w, err)
		return
	}
	actor := actorFromCtx(r)
	if err := h.svc.AssignRole(r.Context(), userID, req.RoleID, actor); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

// RevokeRole DELETE /api/users/{id}/roles/{role_id}
func (h *RBACHandler) RevokeRole(w http.ResponseWriter, r *http.Request) {
	userID := pathID(r)
	roleID, _ := strconv.ParseInt(r.URL.Query().Get("role_id"), 10, 64)
	actor := actorFromCtx(r)
	if err := h.svc.RevokeRole(r.Context(), userID, roleID, actor); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

// GrantPermission POST /api/roles/{id}/permissions
func (h *RBACHandler) GrantPermission(w http.ResponseWriter, r *http.Request) {
	roleID := pathID(r)
	var req struct {
		PermissionID int64 `json:"permission_id"`
	}
	if err := request.Decode(r, h.maxBytes, &req); err != nil {
		response.Error(w, err)
		return
	}
	actor := actorFromCtx(r)
	if err := h.svc.GrantPermission(r.Context(), roleID, req.PermissionID, actor); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}
