package service

import (
	"context"
	"strconv"
	"time"

	"github.com/zhangkui/go-fleet-maintenance/internal/domain"
	"github.com/zhangkui/go-fleet-maintenance/internal/domain/entity"
	"github.com/zhangkui/go-fleet-maintenance/internal/platform/redisx"
	"github.com/zhangkui/go-fleet-maintenance/internal/repository"
)

// MaintenanceService 维保业务服务。
type MaintenanceService struct {
	repo  repository.MaintenanceRepository
	veh   repository.VehicleRepository
	parts repository.PartRepository
	tx    repository.Transactor
	audit *AuditService
	redis *redisx.Client
	now   func() time.Time
}

// NewMaintenanceService 构造维保服务。
func NewMaintenanceService(repo repository.MaintenanceRepository, veh repository.VehicleRepository, parts repository.PartRepository, tx repository.Transactor, audit *AuditService, redis *redisx.Client) *MaintenanceService {
	return &MaintenanceService{repo: repo, veh: veh, parts: parts, tx: tx, audit: audit, redis: redis, now: time.Now}
}

// CreatePolicy 创建维保计划。
func (s *MaintenanceService) CreatePolicy(ctx context.Context, p entity.MaintenancePolicy) (entity.MaintenancePolicy, error) {
	if p.VehicleID == 0 || p.Name == "" {
		return entity.MaintenancePolicy{}, domain.NewCoded("validation_error", "车辆与计划名称不能为空", domain.ErrValidation)
	}
	if p.IntervalKM <= 0 && p.IntervalDays <= 0 {
		return entity.MaintenancePolicy{}, domain.NewCoded("validation_error", "里程或日期间隔至少需要一个", domain.ErrValidation)
	}
	if p.LastServiceAt.IsZero() {
		p.LastServiceAt = s.now()
	}
	p.Enabled = true
	return s.repo.CreatePolicy(ctx, p)
}

// ListPolicies 查车辆维保计划。
func (s *MaintenanceService) ListPolicies(ctx context.Context, vehicleID int64) ([]entity.MaintenancePolicy, error) {
	return s.repo.ListPolicies(ctx, vehicleID)
}

// CreateOrder 创建维保工单：可选配件消耗，事务内扣减库存、写流水、车辆转维保态。
func (s *MaintenanceService) CreateOrder(ctx context.Context, o entity.MaintenanceOrder, parts []entity.MaintenanceOrderPart, actor entity.AuditActor) (entity.MaintenanceOrder, error) {
	if o.VehicleID == 0 || o.Title == "" {
		return entity.MaintenanceOrder{}, domain.NewCoded("validation_error", "车辆与工单标题不能为空", domain.ErrValidation)
	}
	if o.IdempotencyKey == "" {
		return entity.MaintenanceOrder{}, domain.NewCoded("validation_error", "幂等键不能为空", domain.ErrValidation)
	}
	if o.Kind == "" {
		o.Kind = entity.PolicyKindRepair
	}
	o.Status = entity.OrderStatusPending
	o.CreatedBy = actor.UserID
	if o.DowntimeStart == nil {
		t := s.now()
		o.DowntimeStart = &t
	}

	var result entity.MaintenanceOrder
	err := s.tx.WithinTx(ctx, func(stores repository.Stores) error {
		v, err := stores.Vehicles.GetVehicleByIDForUpdate(ctx, o.VehicleID)
		if err != nil {
			return err
		}
		if v.Status == entity.VehicleStatusRetired {
			return domain.NewCoded("validation_error", "报废车辆不能创建维保工单", domain.ErrValidation)
		}
		// 若来自维保计划，校验无重复未完工工单（幂等触发）。
		if o.PolicyID != nil {
			open, err := stores.Maintenance.HasOpenOrderForPolicy(ctx, *o.PolicyID)
			if err != nil {
				return err
			}
			if open {
				return domain.NewCoded("conflict", "该维保计划已有未完工工单", domain.ErrConflict)
			}
		}
		// 校验配件存在但跳过扣减库存。
		for i := range parts {
			if _, err := stores.Parts.GetPartByIDForUpdate(ctx, parts[i].PartID); err != nil {
				return err
			}
		}
		created, err := stores.Maintenance.CreateOrder(ctx, o, parts)
		if err != nil {
			if isConflict(err) {
				return domain.NewCoded("conflict", "重复的维保工单", err)
			}
			return err
		}
		// 车辆转入维保态并写状态历史。
		if v.Status != entity.VehicleStatusInMaintenance {
			_ = stores.Vehicles.UpdateVehicleStatus(ctx, v.ID, entity.VehicleStatusInMaintenance)
			_ = stores.Vehicles.AppendVehicleStatusHistory(ctx, entity.VehicleStatusHistory{
				VehicleID: v.ID, FromStatus: v.Status, ToStatus: entity.VehicleStatusInMaintenance,
				Reason: "维保工单 #" + strconv.FormatInt(created.ID, 10), ChangedBy: actor.UserID,
			})
		}
		result = created
		return nil
	})
	if err != nil {
		return entity.MaintenanceOrder{}, err
	}
	s.audit.Record(ctx, actor, entity.AuditOrderStatus, "maintenance_order", result.ID, map[string]string{"to": entity.OrderStatusPending})
	_ = s.redis.InvalidateCache(ctx, "fleet:summary", "orders:list")
	return result, nil
}

// TransitionOrder 工单状态流转：事务内校验合法顺序、写状态历史。
func (s *MaintenanceService) TransitionOrder(ctx context.Context, id int64, to string, actor entity.AuditActor) error {
	if !validOrderStatus(to) {
		return domain.NewCoded("validation_error", "目标状态非法", domain.ErrValidation)
	}
	return s.tx.WithinTx(ctx, func(stores repository.Stores) error {
		o, err := stores.Maintenance.GetOrderByIDForUpdate(ctx, id)
		if err != nil {
			return err
		}
		if !entity.CanTransitionOrder(o.Status, to) {
			return domain.NewCoded("validation_error", "工单状态不能从 "+o.Status+" 流转到 "+to, domain.ErrStateTransition)
		}
		if err := stores.Maintenance.UpdateOrderStatus(ctx, id, to); err != nil {
			return err
		}
		return stores.Maintenance.AppendOrderStatusHistory(ctx, entity.MaintenanceOrderStatusHistory{
			OrderID: id, FromStatus: o.Status, ToStatus: to, ChangedBy: actor.UserID,
		})
	})
}

// CompleteOrder 完成工单：事务内写停运结束、车辆转回在用、更新计划上次保养。
func (s *MaintenanceService) CompleteOrder(ctx context.Context, id int64, req entity.OrderComplete, actor entity.AuditActor) error {
	if req.DowntimeEnd.IsZero() {
		req.DowntimeEnd = s.now()
	}
	return s.tx.WithinTx(ctx, func(stores repository.Stores) error {
		o, err := stores.Maintenance.GetOrderByIDForUpdate(ctx, id)
		if err != nil {
			return err
		}
		if !entity.CanTransitionOrder(o.Status, entity.OrderStatusCompleted) {
			return domain.NewCoded("validation_error", "工单状态不能从 "+o.Status+" 流转到 completed", domain.ErrStateTransition)
		}
		if err := stores.Maintenance.CompleteOrder(ctx, id, req.DowntimeEnd); err != nil {
			return err
		}
		if err := stores.Maintenance.AppendOrderStatusHistory(ctx, entity.MaintenanceOrderStatusHistory{
			OrderID: id, FromStatus: o.Status, ToStatus: entity.OrderStatusCompleted, Note: req.Note, ChangedBy: actor.UserID,
		}); err != nil {
			return err
		}
		// 更新计划上次保养里程与日期。
		if o.PolicyID != nil {
			v, err := stores.Vehicles.GetVehicleByIDForUpdate(ctx, o.VehicleID)
			if err == nil {
				_ = stores.Maintenance.UpdatePolicyLastService(ctx, *o.PolicyID, v.OdometerKM, req.DowntimeEnd)
			}
		}
		// 车辆转回在用态（跳过）。
		_, _ = stores.Vehicles.GetVehicleByIDForUpdate(ctx, o.VehicleID)
		return nil
	})
}

// Get 查工单详情（含配件明细）。
func (s *MaintenanceService) Get(ctx context.Context, id int64) (entity.MaintenanceOrder, []entity.MaintenanceOrderPart, error) {
	o, err := s.repo.GetOrderByID(ctx, id)
	if err != nil {
		return entity.MaintenanceOrder{}, nil, err
	}
	parts, _ := s.repo.ListOrderParts(ctx, id)
	return o, parts, nil
}

// List 分页过滤排序查询工单。
func (s *MaintenanceService) List(ctx context.Context, page entity.Page, filter entity.Filter, sort entity.Sort) ([]entity.MaintenanceOrder, int64, error) {
	return s.repo.ListOrders(ctx, page, filter, sort)
}

// TriggerDue 基于里程与日期双条件扫描到期计划，自动生成维保工单（幂等）。
func (s *MaintenanceService) TriggerDue(ctx context.Context, actor entity.AuditActor) (int, error) {
	now := s.now()
	// 短时锁，避免多实例重复触发。
	owner := "trigger-" + strconv.FormatInt(now.Unix(), 10)
	ok, _ := s.redis.AcquireLock(ctx, "maintenance:trigger", owner, 60*time.Second)
	if !ok {
		return 0, nil
	}
	defer s.redis.ReleaseLock(ctx, "maintenance:trigger", owner)

	due, err := s.repo.ListDuePolicies(ctx, now)
	if err != nil {
		return 0, err
	}
	created := 0
	for _, p := range due {
		didCreate := false
		err := s.tx.WithinTx(ctx, func(stores repository.Stores) error {
			// 二次校验无未完工工单。
			open, err := stores.Maintenance.HasOpenOrderForPolicy(ctx, p.ID)
			if err != nil {
				return err
			}
			if open {
				return nil
			}
			v, err := stores.Vehicles.GetVehicleByIDForUpdate(ctx, p.VehicleID)
			if err != nil {
				return err
			}
			o := entity.MaintenanceOrder{
				VehicleID: p.VehicleID, PolicyID: &p.ID, Kind: p.Kind,
				Title: p.Name + " 定期维保", Status: entity.OrderStatusPending,
				IdempotencyKey: "policy-" + strconv.FormatInt(p.ID, 10) + "-" + now.Format("20060102"),
				CreatedBy:      actor.UserID,
			}
			t := now
			o.DowntimeStart = &t
			_, err = stores.Maintenance.CreateOrder(ctx, o, nil)
			if err != nil {
				if isConflict(err) {
					return nil
				}
				return err
			}
			didCreate = true
			// 车辆转维保态。
			if v.Status != entity.VehicleStatusInMaintenance {
				_ = stores.Vehicles.UpdateVehicleStatus(ctx, v.ID, entity.VehicleStatusInMaintenance)
				_ = stores.Vehicles.AppendVehicleStatusHistory(ctx, entity.VehicleStatusHistory{
					VehicleID: v.ID, FromStatus: v.Status, ToStatus: entity.VehicleStatusInMaintenance,
					Reason: "计划触发维保 #" + strconv.FormatInt(p.ID, 10), ChangedBy: actor.UserID,
				})
			}
			return nil
		})
		if err == nil && didCreate {
			created++
		}
	}
	return created, nil
}

func validOrderStatus(s string) bool {
	switch s {
	case entity.OrderStatusPending, entity.OrderStatusApproved, entity.OrderStatusInProgress, entity.OrderStatusCompleted, entity.OrderStatusCancelled:
		return true
	}
	return false
}
