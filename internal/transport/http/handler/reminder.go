package handler

import (
	"net/http"
	"strconv"

	"github.com/zhangkui/go-fleet-maintenance/internal/service"
	"github.com/zhangkui/go-fleet-maintenance/internal/transport/http/request"
	"github.com/zhangkui/go-fleet-maintenance/internal/transport/http/response"
)

// ReminderHandler 到期提醒 HTTP 处理器。
type ReminderHandler struct {
	svc      *service.ReminderService
	maxBytes int64
}

// NewReminderHandler 构造提醒处理器。
func NewReminderHandler(svc *service.ReminderService, maxBytes int64) *ReminderHandler {
	return &ReminderHandler{svc: svc, maxBytes: maxBytes}
}

// Scan POST /api/reminders/scan
func (h *ReminderHandler) Scan(w http.ResponseWriter, r *http.Request) {
	lookahead, _ := strconv.Atoi(r.URL.Query().Get("lookahead"))
	result, err := h.svc.Scan(r.Context(), lookahead)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.OK(w, result)
}

// List GET /api/reminders
func (h *ReminderHandler) List(w http.ResponseWriter, r *http.Request) {
	page := request.Page(r)
	filter := request.Filter(r)
	reminders, total, err := h.svc.List(r.Context(), page, filter)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Page(w, reminders, total, page.Limit, page.Offset/page.Limit+1)
}
