// Package response 提供统一 JSON 错误与响应辅助。
package response

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"reflect"

	"github.com/zhangkui/go-fleet-maintenance/internal/domain"
)

// ErrorBody 稳定 JSON 错误结构。
type ErrorBody struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// 写出 JSON。
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("写出 JSON 失败", "err", err)
	}
}

// OK 写出 200。空切片序列化为 []，避免 null。
func OK(w http.ResponseWriter, v interface{}) { writeJSON(w, http.StatusOK, normalize(v)) }

// Created 写出 201。
func Created(w http.ResponseWriter, v interface{}) { writeJSON(w, http.StatusCreated, normalize(v)) }

// NoContent 写出 204。
func NoContent(w http.ResponseWriter) { w.WriteHeader(http.StatusNoContent) }

// Page写出分页结果。空 items 序列化为 []，避免 null。
func Page(w http.ResponseWriter, items interface{}, total int64, limit int, page int) {
	items = normalize(items)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": items,
		"total": total,
		"limit": limit,
		"page":  page,
	})
}

// Error 根据业务错误写出统一 JSON 错误响应。
func Error(w http.ResponseWriter, err error) {
	if err == nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}
	code, status := classify(err)
	writeJSON(w, status, ErrorBody{Code: code, Message: err.Error()})
}

// classify 把领域错误映射为 HTTP 状态码与稳定错误码。
// 优先识别 CodedError 的 code 字段，映射到对应语义状态码。
func classify(err error) (string, int) {
	var ce *domain.CodedError
	if errors.As(err, &ce) {
		return ce.Code, codeToStatus(ce.Code)
	}
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return "not_found", http.StatusNotFound
	case errors.Is(err, domain.ErrConflict), errors.Is(err, domain.ErrIdempotentConflict):
		return "conflict", http.StatusConflict
	case errors.Is(err, domain.ErrValidation), errors.Is(err, domain.ErrMileageNotIncreasing),
		errors.Is(err, domain.ErrStateTransition), errors.Is(err, domain.ErrInvariant):
		return "validation_error", http.StatusBadRequest
	case errors.Is(err, domain.ErrUnauthorized):
		return "unauthorized", http.StatusUnauthorized
	case errors.Is(err, domain.ErrForbidden):
		return "forbidden", http.StatusForbidden
	case errors.Is(err, domain.ErrRateLimited):
		return "rate_limited", http.StatusTooManyRequests
	}
	slog.Error("未分类的业务错误", "err", err)
	return "internal_error", http.StatusInternalServerError
}

// codeToStatus 把 CodedError 的 code 字段映射为 HTTP 状态码。
func codeToStatus(code string) int {
	switch code {
	case "not_found":
		return http.StatusNotFound
	case "conflict":
		return http.StatusConflict
	case "validation_error":
		return http.StatusBadRequest
	case "unauthorized":
		return http.StatusUnauthorized
	case "forbidden":
		return http.StatusForbidden
	case "rate_limited":
		return http.StatusTooManyRequests
	}
	return http.StatusBadRequest
}

// normalize 把 nil 切片归一化为长度 0 的非 nil 切片，使 JSON 输出 [] 而非 null。
func normalize(v interface{}) interface{} {
	if v == nil {
		return v
	}
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Slice && val.IsNil() {
		return reflect.MakeSlice(val.Type(), 0, 0).Interface()
	}
	return v
}
