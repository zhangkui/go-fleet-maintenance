package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/zhangkui/go-fleet-maintenance/internal/domain"
	"github.com/zhangkui/go-fleet-maintenance/internal/repository"
)

var errInvalidID = domain.NewCoded("validation_error", "ID 无效", domain.ErrValidation)

// pathID 从请求路径解析 {id} 路径参数（Go 1.22 ServeMux）。
// 子操作路由如 /api/vehicles/{id}/mileage 中 id 位于中间，必须用 PathValue 提取。
func pathID(r *http.Request) int64 {
	v := r.PathValue("id")
	if v == "" {
		return 0
	}
	var id int64
	if _, err := fmt.Sscanf(v, "%d", &id); err == nil {
		return id
	}
	return 0
}

// PermissionCheckerImpl 基于角色权限码校验。
type PermissionCheckerImpl struct {
	roles repository.RoleRepository
}

// NewPermissionChecker 构造权限校验器。
func NewPermissionChecker(roles repository.RoleRepository) *PermissionCheckerImpl {
	return &PermissionCheckerImpl{roles: roles}
}

// HasPermission 报告用户是否拥有权限码。
func (c *PermissionCheckerImpl) HasPermission(ctx context.Context, userID int64, code string) (bool, error) {
	codes, err := c.roles.UserPermissionCodes(ctx, userID)
	if err != nil {
		return false, err
	}
	for _, code2 := range codes {
		if code2 == code {
			return true, nil
		}
	}
	return false, nil
}

// clientIP 从请求中提取客户端 IP。
func clientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	return r.RemoteAddr
}
