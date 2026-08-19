package service

import (
	"context"
	"time"

	"github.com/zhangkui/go-fleet-maintenance/internal/domain"
	"github.com/zhangkui/go-fleet-maintenance/internal/domain/entity"
	"github.com/zhangkui/go-fleet-maintenance/internal/platform/redisx"
	"github.com/zhangkui/go-fleet-maintenance/internal/repository"
)

// FuelService 油耗业务服务。
type FuelService struct {
	repo  repository.FuelRepository
	veh   repository.VehicleRepository
	tx    repository.Transactor
	audit *AuditService
	redis *redisx.Client
	now   func() time.Time
}

// NewFuelService 构造油耗服务。
func NewFuelService(repo repository.FuelRepository, veh repository.VehicleRepository, tx repository.Transactor, audit *AuditService, redis *redisx.Client) *FuelService {
	return &FuelService{repo: repo, veh: veh, tx: tx, audit: audit, redis: redis, now: time.Now}
}

// Record 录入油耗：校验里程单调递增、计算异常、事务内更新车辆里程、幂等去重。
func (s *FuelService) Record(ctx context.Context, in entity.FuelInput, actor entity.AuditActor) (entity.FuelRecord, error) {
	if in.VehicleID == 0 {
		return entity.FuelRecord{}, domain.NewCoded("validation_error", "车辆不能为空", domain.ErrValidation)
	}
	if in.LitersMilli <= 0 || in.UnitPriceCents <= 0 {
		return entity.FuelRecord{}, domain.NewCoded("validation_error", "加油量与单价必须为正", domain.ErrValidation)
	}
	if in.OdometerKM < 0 {
		return entity.FuelRecord{}, domain.NewCoded("validation_error", "里程不能为负", domain.ErrValidation)
	}
	if in.IdempotencyKey == "" {
		return entity.FuelRecord{}, domain.NewCoded("validation_error", "幂等键不能为空", domain.ErrValidation)
	}
	// 计算总价（分）：毫升 * 分/升 / 1000 / 1000 = 元*分 /...
	// litersMilli(毫升) / 1000 = 升；单价 分/升；总价 = 升 * 单价 = (litersMilli/1000)*unitPriceCents 分。
	// 用整数：totalCents = litersMilli * unitPriceCents / 1000。
	if in.TotalCostCents == 0 {
		product := in.LitersMilli * in.UnitPriceCents
		in.TotalCostCents = product / 1000
		if product%1000 != 0 {
			in.TotalCostCents++
		}
	}
	if in.RecordedAt.IsZero() {
		in.RecordedAt = s.now()
	}

	var result entity.FuelRecord
	err := s.tx.WithinTx(ctx, func(stores repository.Stores) error {
		v, err := stores.Vehicles.GetVehicleByIDForUpdate(ctx, in.VehicleID)
		if err != nil {
			return err
		}
		if in.OdometerKM < v.OdometerKM {
			return domain.ErrMileageNotIncreasing
		}
		// 异常识别：基于最近 N 条相邻里程差计算百公里油耗。
		abnormal := s.detectAbnormal(ctx, stores, in.VehicleID, in.LitersMilli, in.OdometerKM)
		rec := entity.FuelRecord{
			VehicleID: in.VehicleID, LitersMilli: in.LitersMilli, UnitPriceCents: in.UnitPriceCents,
			OdometerKM: in.OdometerKM, TotalCostCents: in.TotalCostCents, Abnormal: abnormal,
			RecordedAt: in.RecordedAt, IdempotencyKey: in.IdempotencyKey, CreatedBy: actor.UserID,
		}
		created, err := stores.Fuel.CreateFuelRecord(ctx, rec)
		if err != nil {
			if isConflict(err) {
				return domain.NewCoded("conflict", "重复的油耗记录", err)
			}
			return err
		}
		// 更新车辆里程。
		if in.OdometerKM > v.OdometerKM {
			if err := stores.Vehicles.UpdateVehicleMileage(ctx, in.VehicleID, in.OdometerKM, in.RecordedAt); err != nil {
				return err
			}
		}
		result = created
		return nil
	})
	if err != nil {
		return entity.FuelRecord{}, err
	}
	s.audit.Record(ctx, actor, entity.AuditFuelRecord, "fuel", result.ID, map[string]int64{"vehicle_id": in.VehicleID, "odometer": in.OdometerKM})
	_ = s.redis.InvalidateCache(ctx, "fleet:summary", "fuel:list")
	return result, nil
}

// detectAbnormal 基于历史百公里油耗均值判断本次是否异常。
func (s *FuelService) detectAbnormal(ctx context.Context, stores repository.Stores, vehicleID int64, litersMilli int64, odometer int64) bool {
	history, err := stores.Fuel.RecentFuelRecords(ctx, vehicleID, 5)
	if err != nil || len(history) == 0 {
		return false
	}
	// 计算历史百公里油耗（升/100km）：相邻记录 (liters_milli/1000) / (odo差) * 100。
	consumptions := make([]float64, 0, len(history))
	for i := 1; i < len(history); i++ {
		delta := history[i].OdometerKM - history[i-1].OdometerKM
		if delta <= 0 {
			continue
		}
		liters := float64(history[i].LitersMilli) / 1000.0
		consumptions = append(consumptions, liters/float64(delta)*100.0)
	}
	if len(consumptions) == 0 {
		return false
	}
	var sum float64
	for _, c := range consumptions {
		sum += c
	}
	avg := sum / float64(len(consumptions))
	if avg <= 0 {
		return false
	}
	lastOdo := history[len(history)-1].OdometerKM
	delta := odometer - lastOdo
	if delta <= 0 {
		return false
	}
	current := float64(litersMilli) / 1000.0 / float64(delta) * 100.0
	// 超过均值 1.5 倍或低于均值 40% 判定异常。
	return current > avg*1.5 || current < avg*0.4
}

// Get 查油耗记录。
func (s *FuelService) Get(ctx context.Context, id int64) (entity.FuelRecord, error) {
	return s.repo.GetFuelRecord(ctx, id)
}

// List 分页过滤排序查询油耗记录。
func (s *FuelService) List(ctx context.Context, page entity.Page, filter entity.Filter, sort entity.Sort) ([]entity.FuelRecord, int64, error) {
	return s.repo.ListFuelRecords(ctx, page, filter, sort)
}
