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

// TripHandler 出车任务 HTTP 处理器。
type TripHandler struct {
	svc      *service.TripService
	maxBytes int64
}

// NewTripHandler 构造任务处理器。
func NewTripHandler(svc *service.TripService, maxBytes int64) *TripHandler {
	return &TripHandler{svc: svc, maxBytes: maxBytes}
}

// Create POST /api/trips
func (h *TripHandler) Create(w http.ResponseWriter, r *http.Request) {
	var t entity.Trip
	if err := request.Decode(r, h.maxBytes, &t); err != nil {
		response.Error(w, err)
		return
	}
	created, err := h.svc.Create(r.Context(), t, actorFromCtx(r))
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, created)
}

// Get GET /api/trips/{id}
func (h *TripHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := pathID(r)
	t, err := h.svc.Get(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.OK(w, t)
}

// List GET /api/trips
func (h *TripHandler) List(w http.ResponseWriter, r *http.Request) {
	page := request.Page(r)
	filter := request.Filter(r)
	sort := request.SortField(r, mysqlrepo.TripSortFields(), "id")
	trips, total, err := h.svc.List(r.Context(), page, filter, sort)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Page(w, trips, total, page.Limit, page.Offset/page.Limit+1)
}

// Start POST /api/trips/{id}/start
func (h *TripHandler) Start(w http.ResponseWriter, r *http.Request) {
	id := pathID(r)
	if err := h.svc.Start(r.Context(), id, actorFromCtx(r)); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

// Complete POST /api/trips/{id}/complete
func (h *TripHandler) Complete(w http.ResponseWriter, r *http.Request) {
	id := pathID(r)
	var req entity.TripComplete
	if err := request.Decode(r, h.maxBytes, &req); err != nil {
		response.Error(w, err)
		return
	}
	if req.CompletedAt.IsZero() {
		req.CompletedAt = time.Now()
	}
	t, err := h.svc.Complete(r.Context(), id, req, actorFromCtx(r))
	if err != nil {
		response.Error(w, err)
		return
	}
	response.OK(w, t)
}

// Cancel POST /api/trips/{id}/cancel
func (h *TripHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	id := pathID(r)
	if err := h.svc.Cancel(r.Context(), id, actorFromCtx(r)); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

// CreateHandover POST /api/trips/{id}/handovers
func (h *TripHandler) CreateHandover(w http.ResponseWriter, r *http.Request) {
	id := pathID(r)
	var hd entity.TripHandover
	if err := request.Decode(r, h.maxBytes, &hd); err != nil {
		response.Error(w, err)
		return
	}
	hd.TripID = id
	if hd.FromDriverID == 0 {
		hd.FromDriverID = 1
	}
	if hd.ToDriverID == 0 {
		hd.ToDriverID = 1
	}
	created, err := h.svc.CreateHandover(r.Context(), hd)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, created)
}
