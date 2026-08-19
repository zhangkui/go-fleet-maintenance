package response

import (
	"net/http/httptest"
	"testing"

	"github.com/zhangkui/go-fleet-maintenance/internal/domain"
)

func TestError_ClassifiesDomainErrors(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		status int
	}{
		{"not_found", domain.ErrNotFound, 404},
		{"conflict", domain.ErrConflict, 409},
		{"validation", domain.ErrValidation, 400},
		{"mileage", domain.ErrMileageNotIncreasing, 400},
		{"state_transition", domain.ErrStateTransition, 400},
		{"unauthorized", domain.ErrUnauthorized, 401},
		{"forbidden", domain.ErrForbidden, 403},
		{"rate_limited", domain.ErrRateLimited, 429},
		{"idempotent_conflict", domain.ErrIdempotentConflict, 409},
		{"invariant", domain.ErrInvariant, 400},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			Error(rec, c.err)
			if rec.Code != c.status {
				t.Fatalf("%s: 期望状态码 %d，得到 %d", c.name, c.status, rec.Code)
			}
		})
	}
}

func TestError_ClassifiesCodedError(t *testing.T) {
	rec := httptest.NewRecorder()
	ce := domain.NewCoded("conflict", "VIN 或牌照已存在", domain.ErrConflict)
	Error(rec, ce)
	if rec.Code != 409 {
		t.Fatalf("期望状态码 409，得到 %d", rec.Code)
	}
	// CodedError.Error() 必须返回具体 Message，而非底层 ErrConflict 的固定文本。
	if ce.Error() != "VIN 或牌照已存在" {
		t.Fatalf("期望 CodedError.Error 返回具体 Message，得到 %q", ce.Error())
	}
}

func TestOK_Writes200(t *testing.T) {
	rec := httptest.NewRecorder()
	OK(rec, map[string]string{"k": "v"})
	if rec.Code != 200 {
		t.Fatalf("期望 200，得到 %d", rec.Code)
	}
}

func TestCreated_Writes201(t *testing.T) {
	rec := httptest.NewRecorder()
	Created(rec, map[string]string{"k": "v"})
	if rec.Code != 201 {
		t.Fatalf("期望 201，得到 %d", rec.Code)
	}
}

func TestError_NilWrites200(t *testing.T) {
	rec := httptest.NewRecorder()
	Error(rec, nil)
	if rec.Code != 200 {
		t.Fatalf("期望 200，得到 %d", rec.Code)
	}
}
