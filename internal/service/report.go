package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/zhangkui/go-fleet-maintenance/internal/domain/entity"
	"github.com/zhangkui/go-fleet-maintenance/internal/platform/redisx"
	"github.com/zhangkui/go-fleet-maintenance/internal/repository"
)

// ReportService 报表服务，热点查询走 Redis 缓存。
type ReportService struct {
	repo  repository.ReportRepository
	redis *redisx.Client
	now   func() time.Time
}

// NewReportService 构造报表服务。
func NewReportService(repo repository.ReportRepository, redis *redisx.Client) *ReportService {
	return &ReportService{repo: repo, redis: redis, now: time.Now}
}

// FleetSummary 车队总览，带 60 秒缓存。
func (s *ReportService) FleetSummary(ctx context.Context) (entity.FleetSummary, error) {
	const cacheKey = "fleet:summary"
	const cacheTTL = 60 * time.Second
	if b, err := s.redis.GetCache(ctx, cacheKey); err == nil && len(b) > 0 {
		var sum entity.FleetSummary
		if json.Unmarshal(b, &sum) == nil {
			return sum, nil
		}
	}
	sum, err := s.repo.FleetSummary(ctx)
	if err != nil {
		return entity.FleetSummary{}, err
	}
	if b, err := json.Marshal(sum); err == nil {
		_ = s.redis.SetCache(ctx, cacheKey, b, cacheTTL)
	}
	return sum, nil
}

// VehicleUtilization 车辆利用率报表。
func (s *ReportService) VehicleUtilization(ctx context.Context, q entity.ReportQuery) ([]entity.VehicleUtilization, int64, error) {
	if q.From.IsZero() {
		q.From = s.now().AddDate(0, -1, 0)
	}
	if q.To.IsZero() {
		q.To = s.now()
	}
	return s.repo.VehicleUtilization(ctx, q.From, q.To, q.Page)
}

// FuelEfficiency 油耗效率报表。
func (s *ReportService) FuelEfficiency(ctx context.Context, q entity.ReportQuery) ([]entity.FuelEfficiencyReport, int64, error) {
	if q.From.IsZero() {
		q.From = s.now()
	}
	if q.To.IsZero() {
		q.To = s.now().AddDate(0, -1, 0)
	}
	return s.repo.FuelEfficiency(ctx, q.From, q.To, q.Page)
}

// MaintenanceCost 维保成本报表。
func (s *ReportService) MaintenanceCost(ctx context.Context, q entity.ReportQuery) ([]entity.MaintenanceCostReport, int64, error) {
	if q.From.IsZero() {
		q.From = s.now().AddDate(0, -1, 0)
	}
	if q.To.IsZero() {
		q.To = s.now()
	}
	return s.repo.MaintenanceCost(ctx, q.From, q.To, q.Page)
}
