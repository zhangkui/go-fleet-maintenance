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

// FuelHandler 油耗 HTTP 处理器。
type FuelHandler struct {
	svc      *service.FuelService
	maxBytes int64
}

// NewFuelHandler 构造油耗处理器。
func NewFuelHandler(svc *service.FuelService, maxBytes int64) *FuelHandler {
	return &FuelHandler{svc: svc, maxBytes: maxBytes}
}

// Record POST /api/fuel-records
func (h *FuelHandler) Record(w http.ResponseWriter, r *http.Request) {
	var in entity.FuelInput
	if err := request.Decode(r, h.maxBytes, &in); err != nil {
		response.Error(w, err)
		return
	}
	if in.RecordedAt.IsZero() {
		in.RecordedAt = time.Now()
	}
	rec, err := h.svc.Record(r.Context(), in, actorFromCtx(r))
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, rec)
}

// Get GET /api/fuel-records/{id}
func (h *FuelHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := pathID(r)
	rec, err := h.svc.Get(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.OK(w, rec)
}

// List GET /api/fuel-records
func (h *FuelHandler) List(w http.ResponseWriter, r *http.Request) {
	page := request.Page(r)
	filter := request.Filter(r)
	sort := request.SortField(r, mysqlrepo.FuelSortFields(), "id")
	records, total, err := h.svc.List(r.Context(), page, filter, sort)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Page(w, records, total, page.Limit, page.Offset/page.Limit+1)
}
