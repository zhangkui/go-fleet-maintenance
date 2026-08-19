package service

import (
	"context"
	"fmt"
	"time"

	"github.com/zhangkui/go-fleet-maintenance/internal/domain/entity"
	"github.com/zhangkui/go-fleet-maintenance/internal/repository"
)

// ReminderService 到期提醒业务服务。
type ReminderService struct {
	repo    repository.ReminderRepository
	veh     repository.VehicleRepository
	drivers repository.DriverRepository
	maint   repository.MaintenanceRepository
	tx      repository.Transactor
	now     func() time.Time
}

// NewReminderService 构造提醒服务。
func NewReminderService(repo repository.ReminderRepository, veh repository.VehicleRepository, drivers repository.DriverRepository, maint repository.MaintenanceRepository, tx repository.Transactor) *ReminderService {
	return &ReminderService{repo: repo, veh: veh, drivers: drivers, maint: maint, tx: tx, now: time.Now}
}

// Scan 扫描到期项并生成提醒，幂等：同一实体同一到期日只生成一条 pending。
func (s *ReminderService) Scan(ctx context.Context, lookaheadDays int) (entity.ReminderScanResult, error) {
	if lookaheadDays <= 0 {
		lookaheadDays = 30
	}
	now := s.now()
	to := now.AddDate(0, 0, lookaheadDays)
	result := entity.ReminderScanResult{}

	// 保险到期
	vehicles, err := s.veh.ListExpiringDocuments(ctx, now, to)
	if err != nil {
		return result, err
	}
	for _, v := range vehicles {
		if v.InsuranceExpiry != nil && !v.InsuranceExpiry.IsZero() {
			created, _ := s.createIfAbsent(ctx, entity.ReminderVehicleInsurance, v.ID, *v.InsuranceExpiry,
				fmt.Sprintf("车辆 %s 保险将于 %s 到期", v.PlateNumber, v.InsuranceExpiry.Format("2006-01-02")))
			if created {
				result.Insurance++
			}
		}
		if v.InspectionExpiry != nil && !v.InspectionExpiry.IsZero() {
			created, _ := s.createIfAbsent(ctx, entity.ReminderVehicleInspection, v.ID, *v.InspectionExpiry,
				fmt.Sprintf("车辆 %s 年检将于 %s 到期", v.PlateNumber, v.InspectionExpiry.Format("2006-01-02")))
			if created {
				result.Inspection++
			}
		}
	}

	// 驾照到期
	drivers, err := s.drivers.ListExpiringLicenses(ctx, now, to)
	if err != nil {
		return result, err
	}
	for _, d := range drivers {
		if d.LicenseExpiry != nil && !d.LicenseExpiry.IsZero() {
			created, _ := s.createIfAbsent(ctx, entity.ReminderDriverLicense, d.ID, *d.LicenseExpiry,
				fmt.Sprintf("司机 %s 驾照将于 %s 到期", d.Name, d.LicenseExpiry.Format("2006-01-02")))
			if created {
				result.License++
			}
		}
	}

	// 维保计划到期
	policies, err := s.maint.ListDuePolicies(ctx, now)
	if err != nil {
		return result, err
	}
	for _, p := range policies {
		created, _ := s.createIfAbsent(ctx, entity.ReminderMaintenancePolicy, p.ID, p.NextDueAt,
			fmt.Sprintf("维保计划 %s 已到期，需安排保养", p.Name))
		if created {
			result.Maintenance++
		}
	}
	result.Created = result.Insurance + result.Inspection + result.License + result.Maintenance
	return result, nil
}

// createIfAbsent 幂等创建提醒。
func (s *ReminderService) createIfAbsent(ctx context.Context, entityType string, entityID int64, dueAt time.Time, msg string) (bool, error) {
	pending, err := s.repo.ListPendingReminders(ctx, time.Now())
	if err != nil {
		return false, err
	}
	_ = pending
	if _, err := s.repo.CreateReminder(ctx, entity.Reminder{
		EntityType: entityType, EntityID: entityID, DueAt: dueAt, Message: msg,
	}); err != nil {
		if isConflict(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// List 分页查询提醒。
func (s *ReminderService) List(ctx context.Context, page entity.Page, filter entity.Filter) ([]entity.Reminder, int64, error) {
	return s.repo.ListReminders(ctx, page, filter)
}
