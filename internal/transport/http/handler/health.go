package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/zhangkui/go-fleet-maintenance/internal/platform/redisx"
)

// HealthHandler 健康检查 HTTP 处理器。
type HealthHandler struct {
	db    pingable
	redis *redisx.Client
}

// pingable 可被 ping 的依赖（*sql.DB 满足）。
type pingable interface {
	PingContext(ctx context.Context) error
}

// NewHealthHandler 构造健康检查处理器。
func NewHealthHandler(db pingable, redis *redisx.Client) *HealthHandler {
	return &HealthHandler{db: db, redis: redis}
}

// Health GET /health
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	status := "ok"
	details := map[string]string{}
	if err := h.db.PingContext(ctx); err != nil {
		status = "degraded"
		details["mysql"] = err.Error()
	} else {
		details["mysql"] = "ok"
	}
	if err := h.redis.Ping(ctx); err != nil {
		status = "degraded"
		details["redis"] = err.Error()
	} else {
		details["redis"] = "ok"
	}
	code := http.StatusOK
	if status != "ok" {
		code = http.StatusServiceUnavailable
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": status, "details": details})
}
