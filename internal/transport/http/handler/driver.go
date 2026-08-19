package handler

import (
	"net/http"
	"time"

	"github.com/zhangkui/go-fleet-maintenance/internal/domain/entity"
	"github.com/zhangkui/go-fleet-maintenance/internal/service"
	"github.com/zhangkui/go-fleet-maintenance/internal/transport/http/request"
	"github.com/zhangkui/go-fleet-maintenance/internal/transport/http/response"
)

// DriverHandler 司机 HTTP 处理器。
type DriverHandler struct {
	svc      *service.DriverService
	maxBytes int64
}

// NewDriverHandler 构造司机处理器。
func NewDriverHandler(svc *service.DriverService, maxBytes int64) *DriverHandler {
	return &DriverHandler{svc: svc, maxBytes: maxBytes}
}

// Create POST /api/drivers
func (h *DriverHandler) Create(w http.ResponseWriter, r *http.Request) {
	var d entity.Driver
	if err := request.Decode(r, h.maxBytes, &d); err != nil {
		response.Error(w, err)
		return
	}
	created, err := h.svc.Create(r.Context(), d)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, created)
}

// Get GET /api/drivers/{id}
func (h *DriverHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := pathID(r)
	d, err := h.svc.Get(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.OK(w, d)
}

// List GET /api/drivers
func (h *DriverHandler) List(w http.ResponseWriter, r *http.Request) {
	page := request.Page(r)
	filter := request.Filter(r)
	drivers, total, err := h.svc.List(r.Context(), page, filter)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Page(w, drivers, total, page.Limit, page.Offset/page.Limit+1)
}

// UpdateStatus POST /api/drivers/{id}/status
func (h *DriverHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id := pathID(r)
	var req struct {
		Status string `json:"status"`
	}
	if err := request.Decode(r, h.maxBytes, &req); err != nil {
		response.Error(w, err)
		return
	}
	if err := h.svc.UpdateStatus(r.Context(), id, req.Status, actorFromCtx(r)); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

// CreateBinding POST /api/drivers/bindings
func (h *DriverHandler) CreateBinding(w http.ResponseWriter, r *http.Request) {
	var b entity.DriverVehicleBinding
	if err := request.Decode(r, h.maxBytes, &b); err != nil {
		response.Error(w, err)
		return
	}
	created, err := h.svc.CreateBinding(r.Context(), b)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, created)
}

// ListBindings GET /api/vehicles/{id}/bindings
func (h *DriverHandler) ListBindings(w http.ResponseWriter, r *http.Request) {
	id := pathID(r)
	bindings, err := h.svc.ListBindings(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.OK(w, bindings)
}

// CreateSchedule POST /api/drivers/schedules
func (h *DriverHandler) CreateSchedule(w http.ResponseWriter, r *http.Request) {
	var s entity.DriverSchedule
	if err := request.Decode(r, h.maxBytes, &s); err != nil {
		response.Error(w, err)
		return
	}
	created, err := h.svc.CreateSchedule(r.Context(), s)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, created)
}

// ListSchedule GET /api/drivers/{id}/schedules
func (h *DriverHandler) ListSchedule(w http.ResponseWriter, r *http.Request) {
	id := pathID(r)
	from, _ := time.Parse(time.RFC3339, r.URL.Query().Get("from"))
	to, _ := time.Parse(time.RFC3339, r.URL.Query().Get("to"))
	sches, err := h.svc.ListSchedule(r.Context(), id, from, to)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.OK(w, sches)
}

// CreateViolation POST /api/drivers/{id}/violations
func (h *DriverHandler) CreateViolation(w http.ResponseWriter, r *http.Request) {
	id := pathID(r)
	var v entity.DriverViolation
	if err := request.Decode(r, h.maxBytes, &v); err != nil {
		response.Error(w, err)
		return
	}
	v.DriverID = id
	created, err := h.svc.CreateViolation(r.Context(), v, actorFromCtx(r))
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, created)
}

// ListViolations GET /api/drivers/{id}/violations
func (h *DriverHandler) ListViolations(w http.ResponseWriter, r *http.Request) {
	id := pathID(r)
	vs, err := h.svc.ListViolations(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.OK(w, vs)
}
