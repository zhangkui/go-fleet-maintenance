package handler

import (
	"net/http"
	"time"

	"github.com/zhangkui/go-fleet-maintenance/internal/domain/entity"
	"github.com/zhangkui/go-fleet-maintenance/internal/service"
	"github.com/zhangkui/go-fleet-maintenance/internal/transport/http/request"
	"github.com/zhangkui/go-fleet-maintenance/internal/transport/http/response"
)

// ReportHandler 报表 HTTP 处理器。
type ReportHandler struct {
	svc      *service.ReportService
	maxBytes int64
}

// NewReportHandler 构造报表处理器。
func NewReportHandler(svc *service.ReportService, maxBytes int64) *ReportHandler {
	return &ReportHandler{svc: svc, maxBytes: maxBytes}
}

func reportQuery(r *http.Request) entity.ReportQuery {
	q := entity.ReportQuery{Page: request.Page(r)}
	q.From, _ = time.Parse(time.RFC3339, r.URL.Query().Get("from"))
	q.To, _ = time.Parse(time.RFC3339, r.URL.Query().Get("to"))
	return q
}

// FleetSummary GET /api/reports/fleet-summary
func (h *ReportHandler) FleetSummary(w http.ResponseWriter, r *http.Request) {
	sum, err := h.svc.FleetSummary(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.OK(w, sum)
}

// VehicleUtilization GET /api/reports/vehicle-utilization
func (h *ReportHandler) VehicleUtilization(w http.ResponseWriter, r *http.Request) {
	q := reportQuery(r)
	items, total, err := h.svc.VehicleUtilization(r.Context(), q)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Page(w, items, total, q.Page.Limit, q.Page.Offset/q.Page.Limit+1)
}

// FuelEfficiency GET /api/reports/fuel-efficiency
func (h *ReportHandler) FuelEfficiency(w http.ResponseWriter, r *http.Request) {
	q := reportQuery(r)
	items, total, err := h.svc.FuelEfficiency(r.Context(), q)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Page(w, items, total, q.Page.Limit, q.Page.Offset/q.Page.Limit+1)
}

// MaintenanceCost GET /api/reports/maintenance-cost
func (h *ReportHandler) MaintenanceCost(w http.ResponseWriter, r *http.Request) {
	q := reportQuery(r)
	items, total, err := h.svc.MaintenanceCost(r.Context(), q)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Page(w, items, total, q.Page.Limit, q.Page.Offset/q.Page.Limit+1)
}
