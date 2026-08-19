// Package request 提供请求解码、校验与公共 DTO。
package request

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/zhangkui/go-fleet-maintenance/internal/domain"
	"github.com/zhangkui/go-fleet-maintenance/internal/domain/entity"
)

// Decode 读取并解码 JSON 请求体，限制最大字节数。
func Decode(r *http.Request, maxBytes int64, dst interface{}) error {
	if r.Body == nil {
		return domain.NewCoded("validation_error", "请求体不能为空", domain.ErrValidation)
	}
	r.Body = http.MaxBytesReader(nil, r.Body, maxBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return domain.NewCoded("validation_error", "请求体不能为空", domain.ErrValidation)
		}
		return domain.NewCoded("validation_error", "请求格式错误: "+err.Error(), domain.ErrValidation)
	}
	return nil
}

// Page 从请求参数解析分页，limit 默认 20、最大 100。
func Page(r *http.Request) entity.Page {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset, _ := strconv.Atoi(q.Get("offset"))
	if offset < 0 {
		offset = 0
	}
	return entity.Page{Limit: limit, Offset: offset}
}

// SortField 从请求参数解析白名单排序字段。
func SortField(r *http.Request, allowed map[string]string, def string) entity.Sort {
	q := r.URL.Query()
	field := q.Get("sort")
	col, ok := allowed[field]
	if !ok || col == "" {
		col = allowed[def]
	}
	order := strings.ToLower(q.Get("order"))
	if order != "asc" && order != "desc" {
		order = "desc"
	}
	return entity.Sort{Field: col, Order: order}
}

// Filter 从请求参数解析通用过滤。
func Filter(r *http.Request) entity.Filter {
	q := r.URL.Query()
	f := entity.Filter{
		Status:  q.Get("status"),
		Keyword: strings.TrimSpace(q.Get("keyword")),
		Kind:    q.Get("kind"),
	}
	if v := q.Get("vehicle_id"); v != "" {
		f.VehicleID, _ = strconv.ParseInt(v, 10, 64)
	}
	if v := q.Get("driver_id"); v != "" {
		f.DriverID, _ = strconv.ParseInt(v, 10, 64)
	}
	if v := q.Get("abnormal"); v != "" {
		b := v == "true" || v == "1"
		f.Abnormal = &b
	}
	if v := q.Get("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			f.From = t
		}
	}
	if v := q.Get("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			f.To = t
		}
	}
	return f
}

// Common DTOs

// LoginRequest 登录请求。
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// RegisterRequest 注册请求。
type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
}

// RefreshRequest 刷新令牌请求。
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// PageReq 分页请求。
type PageReq struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}
