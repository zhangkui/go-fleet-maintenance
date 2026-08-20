package service

import (
	"context"
	"time"

	"github.com/zhangkui/go-fleet-maintenance/internal/domain"
	"github.com/zhangkui/go-fleet-maintenance/internal/domain/entity"
	"github.com/zhangkui/go-fleet-maintenance/internal/platform/redisx"
	"github.com/zhangkui/go-fleet-maintenance/internal/repository"
)

// TripService 出车任务业务服务。
type TripService struct {
	repo  repository.TripRepository
	veh   repository.VehicleRepository
	fuel  repository.FuelRepository
	maint repository.MaintenanceRepository
	tx    repository.Transactor
	audit *AuditService
	redis *redisx.Client
	now   func() time.Time
}

// NewTripService 构造任务服务。
func NewTripService(repo repository.TripRepository, veh repository.VehicleRepository, fuel repository.FuelRepository, maint repository.MaintenanceRepository, tx repository.Transactor, audit *AuditService, redis *redisx.Client) *TripService {
	return &TripService{repo: repo, veh: veh, fuel: fuel, maint: maint, tx: tx, audit: audit, redis: redis, now: time.Now}
}

// Create 创建任务：起点里程不得低于车辆当前里程，幂等键去重。
func (s *TripService) Create(ctx context.Context, t entity.Trip, actor entity.AuditActor) (entity.Trip, error) {
	if t.VehicleID == 0 || t.DriverID == 0 {
		return entity.Trip{}, domain.NewCoded("validation_error", "车辆与司机不能为空", domain.ErrValidation)
	}
	v, err := s.veh.GetVehicleByID(ctx, t.VehicleID)
	if err != nil {
		return entity.Trip{}, err
	}
	if v.Status != entity.VehicleStatusActive {
		return entity.Trip{}, domain.NewCoded("validation_error", "车辆非在用状态，不能派任务", domain.ErrValidation)
	}
	if t.StartOdometerKM < v.OdometerKM {
		return entity.Trip{}, domain.ErrMileageNotIncreasing
	}
	if t.IdempotencyKey == "" {
		return entity.Trip{}, domain.NewCoded("validation_error", "幂等键不能为空", domain.ErrValidation)
	}
	// Redis 短期幂等标记，数据库唯一约束兜底。
	first, err := s.redis.SetIdempotency(ctx, "trip:"+t.IdempotencyKey, t.IdempotencyKey, 24*time.Hour)
	if err != nil {
		return entity.Trip{}, err
	}
	if !first {
		// 重复请求：幂等返回既有任务（按幂等键查最近一条）。
		rows, _, _ := s.repo.ListTrips(ctx, entity.Page{Limit: 1, Offset: 0}, entity.Filter{}, entity.Sort{})
		if len(rows) > 0 {
			return rows[0], nil
		}
	}
	t.Status = entity.TripStatusScheduled
	created, err := s.repo.CreateTrip(ctx, t)
	if err != nil {
		if isConflict(err) {
			// 幂等键重复：返回既有任务。
			return s.repo.GetTripByID(ctx, t.ID)
		}
		return entity.Trip{}, err
	}
	return created, nil
}

// Get 查任务详情。
func (s *TripService) Get(ctx context.Context, id int64) (entity.Trip, error) {
	return s.repo.GetTripByID(ctx, id)
}

// List 分页过滤排序查询任务。
func (s *TripService) List(ctx context.Context, page entity.Page, filter entity.Filter, sort entity.Sort) ([]entity.Trip, int64, error) {
	return s.repo.ListTrips(ctx, page, filter, sort)
}

// Start 开始任务：scheduled -> in_progress，事务化写状态历史。
func (s *TripService) Start(ctx context.Context, id int64, actor entity.AuditActor) error {
	return s.tx.WithinTx(ctx, func(stores repository.Stores) error {
		t, err := stores.Trips.GetTripByIDForUpdate(ctx, id)
		if err != nil {
			return err
		}
		if !entity.CanTransitionTrip(t.Status, entity.TripStatusInProgress) {
			return domain.NewCoded("validation_error", "任务状态不能从 "+t.Status+" 流转到 in_progress", domain.ErrStateTransition)
		}
		if err := stores.Trips.StartTrip(ctx, id, actor.UserID, s.now().Add(-24*time.Hour)); err != nil {
			return err
		}
		return stores.Trips.AppendTripStatusHistory(ctx, entity.TripStatusHistory{
			TripID: id, FromStatus: t.Status, ToStatus: entity.TripStatusInProgress, ChangedBy: actor.UserID,
		})
	})
}

// Complete 完成任务：事务内更新车辆里程、完成任务、写状态历史、触发维保检查。
// 里程单调递增、任务完结与里程更新必须在同一事务中保持一致。
func (s *TripService) Complete(ctx context.Context, id int64, req entity.TripComplete, actor entity.AuditActor) (entity.Trip, error) {
	if req.EndOdometerKM < 0 {
		return entity.Trip{}, domain.NewCoded("validation_error", "结束里程不能为负", domain.ErrValidation)
	}
	var result entity.Trip
	err := s.tx.WithinTx(ctx, func(stores repository.Stores) error {
		t, err := stores.Trips.GetTripByIDForUpdate(ctx, id)
		if err != nil {
			return err
		}
		if !entity.CanTransitionTrip(t.Status, entity.TripStatusCompleted) {
			return domain.NewCoded("validation_error", "任务状态不能从 "+t.Status+" 流转到 completed", domain.ErrStateTransition)
		}
		if req.EndOdometerKM < t.StartOdometerKM {
			return domain.ErrMileageNotIncreasing
		}
		// 加行锁读车辆，校验里程单调递增后更新。
		v, err := stores.Vehicles.GetVehicleByIDForUpdate(ctx, t.VehicleID)
		if err != nil {
			return err
		}
		if req.EndOdometerKM < v.OdometerKM {
			return domain.ErrMileageNotIncreasing
		}
		if err := stores.Trips.CompleteTrip(ctx, id, req.EndOdometerKM, actor.UserID, req.CompletedAt); err != nil {
			return err
		}
		if req.EndOdometerKM > v.OdometerKM {
			if err := stores.Vehicles.UpdateVehicleMileage(ctx, t.VehicleID, req.EndOdometerKM, req.CompletedAt); err != nil {
				return err
			}
		}
		if err := stores.Trips.AppendTripStatusHistory(ctx, entity.TripStatusHistory{
			TripID: id, FromStatus: t.Status, ToStatus: entity.TripStatusCompleted, Note: req.Note, ChangedBy: actor.UserID,
		}); err != nil {
			return err
		}
		t.Status = entity.TripStatusCompleted
		edo := req.EndOdometerKM
		t.EndOdometerKM = &edo
		result = t
		return nil
	})
	if err != nil {
		return entity.Trip{}, err
	}
	s.audit.Record(ctx, actor, entity.AuditTripComplete, "trip", id, map[string]int64{"end_odometer": req.EndOdometerKM})
	_ = s.redis.InvalidateCache(ctx, "fleet:summary", "trips:list")
	return result, nil
}

// Cancel 取消任务。
func (s *TripService) Cancel(ctx context.Context, id int64, actor entity.AuditActor) error {
	return s.tx.WithinTx(ctx, func(stores repository.Stores) error {
		t, err := stores.Trips.GetTripByIDForUpdate(ctx, id)
		if err != nil {
			return err
		}
		if !entity.CanTransitionTrip(t.Status, entity.TripStatusCancelled) {
			return domain.NewCoded("validation_error", "任务状态不能从 "+t.Status+" 流转到 cancelled", domain.ErrStateTransition)
		}
		if err := stores.Trips.CancelTrip(ctx, id, actor.UserID, s.now()); err != nil {
			return err
		}
		return stores.Trips.AppendTripStatusHistory(ctx, entity.TripStatusHistory{
			TripID: id, FromStatus: t.Status, ToStatus: entity.TripStatusCancelled, ChangedBy: actor.UserID,
		})
	})
}

// CreateHandover 创建任务交接。
func (s *TripService) CreateHandover(ctx context.Context, h entity.TripHandover) (entity.TripHandover, error) {
	if h.TripID == 0 {
		return entity.TripHandover{}, domain.NewCoded("validation_error", "任务不能为空", domain.ErrValidation)
	}
	if h.HandoverAt.IsZero() {
		h.HandoverAt = s.now()
	}
	return s.repo.CreateHandover(ctx, h)
}
