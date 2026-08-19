package handler

import (
	"net/http"

	"github.com/zhangkui/go-fleet-maintenance/internal/domain/entity"
	"github.com/zhangkui/go-fleet-maintenance/internal/transport/http/middleware"
)

// actorFromCtx 从请求上下文取出操作者。
func actorFromCtx(r *http.Request) entity.AuditActor {
	return middleware.ActorFromContext(r.Context())
}
