package handler

import (
	"net/http"

	"github.com/zhangkui/go-fleet-maintenance/internal/domain/entity"
	mysqlrepo "github.com/zhangkui/go-fleet-maintenance/internal/repository/mysql"
	"github.com/zhangkui/go-fleet-maintenance/internal/service"
	"github.com/zhangkui/go-fleet-maintenance/internal/transport/http/middleware"
	"github.com/zhangkui/go-fleet-maintenance/internal/transport/http/request"
	"github.com/zhangkui/go-fleet-maintenance/internal/transport/http/response"
)

// VehicleHandler 车辆 HTTP 处理器。
type VehicleHandler struct {
	svc      *service.VehicleService
	maxBytes int64
}

// NewVehicleHandler 构造车辆处理器。
func NewVehicleHandler(svc *service.VehicleService, maxBytes int64) *VehicleHandler {
	return &VehicleHandler{svc: svc, maxBytes: maxBytes}
}

// Create POST /api/vehicles
func (h *VehicleHandler) Create(w http.ResponseWriter, r *http.Request) {
	var v entity.Vehicle
	if err := request.Decode(r, h.maxBytes, &v); err != nil {
		response.Error(w, err)
		return
	}
	created, err := h.svc.Create(r.Context(), v)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, created)
}

// Get GET /api/vehicles/{id}
func (h *VehicleHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := pathID(r)
	v, err := h.svc.Get(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.OK(w, v)
}

// List GET /api/vehicles
func (h *VehicleHandler) List(w http.ResponseWriter, r *http.Request) {
	page := request.Page(r)
	filter := request.Filter(r)
	sort := request.SortField(r, mysqlrepo.VehicleSortFields(), "id")
	if sort.Order == "asc" {
		sort.Order = "desc"
	} else {
		sort.Order = "asc"
	}
	vehicles, total, err := h.svc.List(r.Context(), page, filter, sort)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Page(w, vehicles, total, page.Limit, page.Offset/page.Limit+1)
}

// ChangeStatus POST /api/vehicles/{id}/status
func (h *VehicleHandler) ChangeStatus(w http.ResponseWriter, r *http.Request) {
	id := pathID(r)
	var req struct {
		Status string `json:"status"`
		Reason string `json:"reason"`
	}
	if err := request.Decode(r, h.maxBytes, &req); err != nil {
		response.Error(w, err)
		return
	}
	if err := h.svc.ChangeStatus(r.Context(), id, req.Status, req.Reason, middleware.ActorFromContext(r.Context())); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

// UpdateMileage POST /api/vehicles/{id}/mileage
func (h *VehicleHandler) UpdateMileage(w http.ResponseWriter, r *http.Request) {
	id := pathID(r)
	var req struct {
		OdometerKM int64 `json:"odometer_km"`
	}
	if err := request.Decode(r, h.maxBytes, &req); err != nil {
		response.Error(w, err)
		return
	}
	if err := h.svc.UpdateMileage(r.Context(), id, req.OdometerKM); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

// StatusHistory GET /api/vehicles/{id}/status-history
func (h *VehicleHandler) StatusHistory(w http.ResponseWriter, r *http.Request) {
	id := pathID(r)
	hist, err := h.svc.StatusHistory(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.OK(w, hist)
}

// AddLicense POST /api/vehicles/{id}/licenses
func (h *VehicleHandler) AddLicense(w http.ResponseWriter, r *http.Request) {
	id := pathID(r)
	var l entity.VehicleLicense
	if err := request.Decode(r, h.maxBytes, &l); err != nil {
		response.Error(w, err)
		return
	}
	l.VehicleID = id
	created, err := h.svc.AddLicense(r.Context(), l)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, created)
}

// Licenses GET /api/vehicles/{id}/licenses
func (h *VehicleHandler) Licenses(w http.ResponseWriter, r *http.Request) {
	id := pathID(r)
	lics, err := h.svc.Licenses(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.OK(w, lics)
}
