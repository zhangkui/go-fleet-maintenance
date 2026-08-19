package handler

import (
	"net/http"

	"github.com/zhangkui/go-fleet-maintenance/internal/domain/entity"
	"github.com/zhangkui/go-fleet-maintenance/internal/service"
	"github.com/zhangkui/go-fleet-maintenance/internal/transport/http/request"
	"github.com/zhangkui/go-fleet-maintenance/internal/transport/http/response"
)

// PartHandler 配件 HTTP 处理器。
type PartHandler struct {
	svc      *service.PartService
	maxBytes int64
}

// NewPartHandler 构造配件处理器。
func NewPartHandler(svc *service.PartService, maxBytes int64) *PartHandler {
	return &PartHandler{svc: svc, maxBytes: maxBytes}
}

// Create POST /api/parts
func (h *PartHandler) Create(w http.ResponseWriter, r *http.Request) {
	var p entity.Part
	if err := request.Decode(r, h.maxBytes, &p); err != nil {
		response.Error(w, err)
		return
	}
	created, err := h.svc.Create(r.Context(), p)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, created)
}

// List GET /api/parts
func (h *PartHandler) List(w http.ResponseWriter, r *http.Request) {
	page := request.Page(r)
	filter := request.Filter(r)
	parts, total, err := h.svc.List(r.Context(), page, filter)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Page(w, parts, total, page.Limit, page.Offset/page.Limit+1)
}

// ListLowStock GET /api/parts/low-stock
func (h *PartHandler) ListLowStock(w http.ResponseWriter, r *http.Request) {
	parts, err := h.svc.ListLowStock(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.OK(w, parts)
}

// AdjustStock POST /api/parts/{id}/adjust
func (h *PartHandler) AdjustStock(w http.ResponseWriter, r *http.Request) {
	id := pathID(r)
	var req entity.PartAdjust
	if err := request.Decode(r, h.maxBytes, &req); err != nil {
		response.Error(w, err)
		return
	}
	req.PartID = id
	p, err := h.svc.AdjustStock(r.Context(), req, actorFromCtx(r))
	if err != nil {
		response.Error(w, err)
		return
	}
	response.OK(w, p)
}

// ListStockMovements GET /api/parts/{id}/movements
func (h *PartHandler) ListStockMovements(w http.ResponseWriter, r *http.Request) {
	id := pathID(r)
	page := request.Page(r)
	movs, total, err := h.svc.ListStockMovements(r.Context(), id, page)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Page(w, movs, total, page.Limit, page.Offset/page.Limit+1)
}
