package service

import (
	"context"
	"strings"
	"time"

	"github.com/zhangkui/go-fleet-maintenance/internal/domain"
	"github.com/zhangkui/go-fleet-maintenance/internal/domain/entity"
	"github.com/zhangkui/go-fleet-maintenance/internal/platform/redisx"
	"github.com/zhangkui/go-fleet-maintenance/internal/repository"
)

// VehicleService 车辆业务服务。
type VehicleService struct {
	repo  repository.VehicleRepository
	tx    repository.Transactor
	audit *AuditService
	redis *redisx.Client
	now   func() time.Time
}

// NewVehicleService 构造车辆服务。
func NewVehicleService(repo repository.VehicleRepository, tx repository.Transactor, audit *AuditService, redis *redisx.Client) *VehicleService {
	return &VehicleService{repo: repo, tx: tx, audit: audit, redis: redis, now: time.Now}
}

// Create 创建车辆：规范化 VIN/牌照，状态默认 active。
func (s *VehicleService) Create(ctx context.Context, v entity.Vehicle) (entity.Vehicle, error) {
	v.VIN = strings.ToUpper(strings.TrimSpace(v.VIN))
	v.PlateNumber = strings.ToUpper(strings.TrimSpace(v.PlateNumber))
	v.Model = strings.TrimSpace(v.Model)
	if v.VIN == "" || v.PlateNumber == "" || v.Model == "" {
		return entity.Vehicle{}, domain.NewCoded("validation_error", "车型、VIN 和牌照不能为空", domain.ErrValidation)
	}
	if v.Status == "" {
		v.Status = entity.VehicleStatusActive
	}
	if !validVehicleStatus(v.Status) {
		return entity.Vehicle{}, domain.NewCoded("validation_error", "车辆状态非法", domain.ErrValidation)
	}
	created, err := s.repo.CreateVehicle(ctx, v)
	if err != nil {
		if isConflict(err) {
			return entity.Vehicle{}, domain.NewCoded("conflict", "VIN 或牌照已存在", err)
		}
		return entity.Vehicle{}, err
	}
	s.invalidateVehicleCache(ctx)
	return created, nil
}

// Get 查车辆详情。
func (s *VehicleService) Get(ctx context.Context, id int64) (entity.Vehicle, error) {
	return s.repo.GetVehicleByID(ctx, id)
}

// List 分页过滤排序查询车辆。
func (s *VehicleService) List(ctx context.Context, page entity.Page, filter entity.Filter, sort entity.Sort) ([]entity.Vehicle, int64, error) {
	return s.repo.ListVehicles(ctx, page, filter, sort)
}

// ChangeStatus 车辆状态流转：事务内校验合法顺序、写状态历史。
func (s *VehicleService) ChangeStatus(ctx context.Context, id int64, to string, reason string, actor entity.AuditActor) error {
	if !validVehicleStatus(to) {
		return domain.NewCoded("validation_error", "目标状态非法", domain.ErrValidation)
	}
	return s.tx.WithinTx(ctx, func(stores repository.Stores) error {
		v, err := stores.Vehicles.GetVehicleByIDForUpdate(ctx, id)
		if err != nil {
			return err
		}
		if !entity.CanTransitionVehicle(v.Status, to) {
			return domain.NewCoded("validation_error",
				"车辆状态不能从 "+v.Status+" 流转到 "+to, domain.ErrStateTransition)
		}
		if err := stores.Vehicles.UpdateVehicleStatus(ctx, id, to); err != nil {
			return err
		}
		if err := stores.Vehicles.AppendVehicleStatusHistory(ctx, entity.VehicleStatusHistory{
			VehicleID: id, FromStatus: to, ToStatus: to, Reason: reason, ChangedBy: actor.UserID,
		}); err != nil {
			return err
		}
		return nil
	})
}

// UpdateMileage 更新车辆里程：单调递增校验。
func (s *VehicleService) UpdateMileage(ctx context.Context, id, odometer int64) error {
	if odometer < 0 {
		return domain.NewCoded("validation_error", "里程不能为负", domain.ErrValidation)
	}
	v, err := s.repo.GetVehicleByID(ctx, id)
	if err != nil {
		return err
	}
	if odometer <= v.OdometerKM {
		return domain.ErrMileageNotIncreasing
	}
	if err := s.repo.UpdateVehicleMileage(ctx, id, odometer, s.now().Add(-24*time.Hour)); err != nil {
		return err
	}
	s.invalidateVehicleCache(ctx)
	return nil
}

// StatusHistory 查车辆状态流转。
func (s *VehicleService) StatusHistory(ctx context.Context, vehicleID int64) ([]entity.VehicleStatusHistory, error) {
	return s.repo.ListVehicleStatusHistory(ctx, vehicleID)
}

// AddLicense 添加车辆证照。
func (s *VehicleService) AddLicense(ctx context.Context, l entity.VehicleLicense) (entity.VehicleLicense, error) {
	if l.VehicleID == 0 || l.Kind == "" {
		return entity.VehicleLicense{}, domain.NewCoded("validation_error", "车辆 ID 与证照类型不能为空", domain.ErrValidation)
	}
	if l.ExpiryDate == nil || l.ExpiryDate.IsZero() {
		return entity.VehicleLicense{}, domain.NewCoded("validation_error", "证照到期日不能为空", domain.ErrValidation)
	}
	created, err := s.repo.CreateVehicleLicense(ctx, l)
	if err != nil {
		return entity.VehicleLicense{}, err
	}
	// 同步更新车辆保险/年检到期日，便于到期扫描。
	_ = s.repo.UpdateVehicleLicenseExpiry(ctx, l.VehicleID, l.Kind, *l.ExpiryDate)
	s.invalidateVehicleCache(ctx)
	return created, nil
}

// Licenses 查车辆证照。
func (s *VehicleService) Licenses(ctx context.Context, vehicleID int64) ([]entity.VehicleLicense, error) {
	return s.repo.ListVehicleLicenses(ctx, vehicleID)
}

// invalidateVehicleCache 失效车辆相关缓存。
func (s *VehicleService) invalidateVehicleCache(ctx context.Context) {
	_ = s.redis.InvalidateCache(ctx, "fleet:summary", "vehicles:list")
}

func validVehicleStatus(s string) bool {
	switch s {
	case entity.VehicleStatusActive, entity.VehicleStatusInMaintenance, entity.VehicleStatusRetired:
		return true
	}
	return false
}

func isConflict(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "Duplicate entry") || err == domain.ErrConflict)
}
