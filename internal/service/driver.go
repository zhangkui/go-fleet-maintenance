package service

import (
	"context"
	"strings"
	"time"

	"github.com/zhangkui/go-fleet-maintenance/internal/domain"
	"github.com/zhangkui/go-fleet-maintenance/internal/domain/entity"
	"github.com/zhangkui/go-fleet-maintenance/internal/repository"
)

// DriverService 司机业务服务。
type DriverService struct {
	repo  repository.DriverRepository
	audit *AuditService
	now   func() time.Time
}

// NewDriverService 构造司机服务。
func NewDriverService(repo repository.DriverRepository, audit *AuditService) *DriverService {
	return &DriverService{repo: repo, audit: audit, now: time.Now}
}

// Create 创建司机：规范化驾照号。
func (s *DriverService) Create(ctx context.Context, d entity.Driver) (entity.Driver, error) {
	d.Name = strings.TrimSpace(d.Name)
	d.LicenseNumber = strings.ToUpper(strings.TrimSpace(d.LicenseNumber))
	if d.Name == "" || d.LicenseNumber == "" || d.LicenseClass == "" {
		return entity.Driver{}, domain.NewCoded("validation_error", "姓名、驾照号和准驾车型不能为空", domain.ErrValidation)
	}
	if d.LicenseExpiry == nil || d.LicenseExpiry.IsZero() {
		return entity.Driver{}, domain.NewCoded("validation_error", "驾照到期日不能为空", domain.ErrValidation)
	}
	d.Status = entity.DriverStatusActive
	created, err := s.repo.CreateDriver(ctx, d)
	if err != nil {
		if isConflict(err) {
			return entity.Driver{}, domain.NewCoded("conflict", "驾照号已存在", err)
		}
		return entity.Driver{}, err
	}
	return created, nil
}

// Get 查司机详情。
func (s *DriverService) Get(ctx context.Context, id int64) (entity.Driver, error) {
	return s.repo.GetDriverByID(ctx, id)
}

// List 分页过滤查询司机。
func (s *DriverService) List(ctx context.Context, page entity.Page, filter entity.Filter) ([]entity.Driver, int64, error) {
	return s.repo.ListDrivers(ctx, page, filter)
}

// UpdateStatus 更新司机状态。
func (s *DriverService) UpdateStatus(ctx context.Context, id int64, status string, actor entity.AuditActor) error {
	switch status {
	case entity.DriverStatusActive, entity.DriverStatusSuspended, entity.DriverStatusResigned:
	default:
		return domain.NewCoded("validation_error", "司机状态非法", domain.ErrValidation)
	}
	if err := s.repo.UpdateDriverStatus(ctx, id, status); err != nil {
		return err
	}
	return nil
}

// CreateBinding 创建司机-车辆绑定：校验车辆无冲突活跃绑定。
func (s *DriverService) CreateBinding(ctx context.Context, b entity.DriverVehicleBinding) (entity.DriverVehicleBinding, error) {
	if b.DriverID == 0 || b.VehicleID == 0 {
		return entity.DriverVehicleBinding{}, domain.NewCoded("validation_error", "司机与车辆不能为空", domain.ErrValidation)
	}
	if b.StartDate.IsZero() {
		b.StartDate = s.now()
	}
	exists, err := s.repo.HasActiveBindingForVehicle(ctx, b.VehicleID, s.now())
	if err != nil {
		return entity.DriverVehicleBinding{}, err
	}
	if exists {
		return entity.DriverVehicleBinding{}, domain.NewCoded("conflict", "该车辆在该时段已有活跃绑定", domain.ErrConflict)
	}
	b.Status = entity.BindingStatusActive
	return s.repo.CreateBinding(ctx, b)
}

// ListBindings 查车辆活跃绑定。
func (s *DriverService) ListBindings(ctx context.Context, vehicleID int64) ([]entity.DriverVehicleBinding, error) {
	return s.repo.ListActiveBindings(ctx, vehicleID, s.now())
}

// CreateSchedule 创建排班。
func (s *DriverService) CreateSchedule(ctx context.Context, sch entity.DriverSchedule) (entity.DriverSchedule, error) {
	if sch.DriverID == 0 || sch.ShiftDate.IsZero() {
		return entity.DriverSchedule{}, domain.NewCoded("validation_error", "司机与班次日期不能为空", domain.ErrValidation)
	}
	switch sch.ShiftType {
	case entity.ShiftMorning, entity.ShiftEvening, entity.ShiftNight:
	default:
		return entity.DriverSchedule{}, domain.NewCoded("validation_error", "班次类型非法", domain.ErrValidation)
	}
	return s.repo.CreateSchedule(ctx, sch)
}

// ListSchedule 查司机排班。
func (s *DriverService) ListSchedule(ctx context.Context, driverID int64, from, to time.Time) ([]entity.DriverSchedule, error) {
	if from.IsZero() {
		from = s.now().AddDate(0, 0, -30)
	}
	if to.IsZero() {
		to = s.now().AddDate(0, 0, 30)
	}
	return s.repo.ListScheduleByDriver(ctx, driverID, from, to)
}

// CreateViolation 创建违章记录。
func (s *DriverService) CreateViolation(ctx context.Context, v entity.DriverViolation, actor entity.AuditActor) (entity.DriverViolation, error) {
	if v.DriverID == 0 || v.ViolationType == "" {
		return entity.DriverViolation{}, domain.NewCoded("validation_error", "司机与违章类型不能为空", domain.ErrValidation)
	}
	if v.OccurredAt.IsZero() {
		v.OccurredAt = s.now()
	}
	created, err := s.repo.CreateViolation(ctx, v)
	if err != nil {
		return entity.DriverViolation{}, err
	}
	s.audit.Record(ctx, actor, "driver.violation_create", "driver", v.DriverID, map[string]int64{"violation_id": created.ID})
	return created, nil
}

// ListViolations 查司机违章。
func (s *DriverService) ListViolations(ctx context.Context, driverID int64) ([]entity.DriverViolation, error) {
	return s.repo.ListViolations(ctx, driverID)
}
