package handler

import (
	"net/http"
	"time"

	"github.com/zhangkui/go-fleet-maintenance/internal/domain/entity"
	mysqlrepo "github.com/zhangkui/go-fleet-maintenance/internal/repository/mysql"
	"github.com/zhangkui/go-fleet-maintenance/internal/service"
	"github.com/zhangkui/go-fleet-maintenance/internal/transport/http/request"
	"github.com/zhangkui/go-fleet-maintenance/internal/transport/http/response"
)

// MaintenanceHandler 维保 HTTP 处理器。
type MaintenanceHandler struct {
	svc      *service.MaintenanceService
	maxBytes int64
}

// NewMaintenanceHandler 构造维保处理器。
func NewMaintenanceHandler(svc *service.MaintenanceService, maxBytes int64) *MaintenanceHandler {
	return &MaintenanceHandler{svc: svc, maxBytes: maxBytes}
}

// CreatePolicy POST /api/maintenance/policies
func (h *MaintenanceHandler) CreatePolicy(w http.ResponseWriter, r *http.Request) {
	var p entity.MaintenancePolicy
	if err := request.Decode(r, h.maxBytes, &p); err != nil {
		response.Error(w, err)
		return
	}
	created, err := h.svc.CreatePolicy(r.Context(), p)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, created)
}

// ListPolicies GET /api/maintenance/policies?vehicle_id=
func (h *MaintenanceHandler) ListPolicies(w http.ResponseWriter, r *http.Request) {
	vehicleID := pathID(r)
	policies, err := h.svc.ListPolicies(r.Context(), vehicleID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.OK(w, policies)
}

// CreateOrderInput 创建工单请求体。
type CreateOrderInput struct {
	entity.MaintenanceOrder
	Parts []entity.MaintenanceOrderPart `json:"parts"`
}

// CreateOrder POST /api/maintenance/orders
func (h *MaintenanceHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var in CreateOrderInput
	if err := request.Decode(r, h.maxBytes, &in); err != nil {
		response.Error(w, err)
		return
	}
	if in.DowntimeStart == nil {
		t := time.Now()
		in.DowntimeStart = &t
	}
	created, err := h.svc.CreateOrder(r.Context(), in.MaintenanceOrder, in.Parts, actorFromCtx(r))
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, created)
}

// GetOrder GET /api/maintenance/orders/{id}
func (h *MaintenanceHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	id := pathID(r)
	o, parts, err := h.svc.Get(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.OK(w, map[string]interface{}{"order": o, "parts": parts})
}

// ListOrders GET /api/maintenance/orders
func (h *MaintenanceHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	page := request.Page(r)
	filter := request.Filter(r)
	sort := request.SortField(r, mysqlrepo.OrderSortFields(), "id")
	orders, total, err := h.svc.List(r.Context(), page, filter, sort)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Page(w, orders, total, page.Limit, page.Offset/page.Limit+1)
}

// TransitionOrder POST /api/maintenance/orders/{id}/status
func (h *MaintenanceHandler) TransitionOrder(w http.ResponseWriter, r *http.Request) {
	id := pathID(r)
	var req struct {
		Status string `json:"status"`
	}
	if err := request.Decode(r, h.maxBytes, &req); err != nil {
		response.Error(w, err)
		return
	}
	if err := h.svc.TransitionOrder(r.Context(), id, req.Status, actorFromCtx(r)); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

// CompleteOrder POST /api/maintenance/orders/{id}/complete
func (h *MaintenanceHandler) CompleteOrder(w http.ResponseWriter, r *http.Request) {
	id := pathID(r)
	var req entity.OrderComplete
	if err := request.Decode(r, h.maxBytes, &req); err != nil {
		response.Error(w, err)
		return
	}
	if req.DowntimeEnd.IsZero() {
		req.DowntimeEnd = time.Now()
	}
	if err := h.svc.CompleteOrder(r.Context(), id, req, actorFromCtx(r)); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

// TriggerDue POST /api/maintenance/trigger-due
func (h *MaintenanceHandler) TriggerDue(w http.ResponseWriter, r *http.Request) {
	n, err := h.svc.TriggerDue(r.Context(), actorFromCtx(r))
	if err != nil {
		response.Error(w, err)
		return
	}
	response.OK(w, map[string]int{"created": n})
}
