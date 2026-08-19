package service

import (
	"context"
	"errors"
	"testing"

	"github.com/zhangkui/go-fleet-maintenance/internal/domain"
	"github.com/zhangkui/go-fleet-maintenance/internal/domain/entity"
	"github.com/zhangkui/go-fleet-maintenance/internal/repository"
)

// fakeTransactor 事务替身：直接复用 stores 并执行 fn。
type fakeTransactor struct {
	stores repository.Stores
}

func (t *fakeTransactor) WithinTx(ctx context.Context, fn func(repository.Stores) error) error {
	return fn(t.stores)
}

func TestVehicleService_Create_NormalizesVINAndPlate(t *testing.T) {
	vehRepo := newFakeVehicleRepo()
	tx := &fakeTransactor{stores: repository.Stores{Vehicles: vehRepo}}
	auditSvc := &AuditService{repo: &nopAuditRepo{}}
	svc := NewVehicleService(vehRepo, tx, auditSvc, nil)

	v, err := svc.Create(context.Background(), entity.Vehicle{Model: "东风", VIN: " vin001 ", PlateNumber: " 京a123 ", Color: "白"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if v.VIN != "VIN001" {
		t.Fatalf("VIN 未规范化大写: %q", v.VIN)
	}
	if v.PlateNumber != "京A123" {
		t.Fatalf("牌照未规范化大写: %q", v.PlateNumber)
	}
	if v.Status != entity.VehicleStatusActive {
		t.Fatalf("默认状态非 active: %q", v.Status)
	}
}

func TestVehicleService_Create_RejectsDuplicateVIN(t *testing.T) {
	vehRepo := newFakeVehicleRepo()
	tx := &fakeTransactor{stores: repository.Stores{Vehicles: vehRepo}}
	auditSvc := &AuditService{repo: &nopAuditRepo{}}
	svc := NewVehicleService(vehRepo, tx, auditSvc, nil)

	_, _ = svc.Create(context.Background(), entity.Vehicle{Model: "A", VIN: "VIN1", PlateNumber: "P1"})
	_, err := svc.Create(context.Background(), entity.Vehicle{Model: "B", VIN: "VIN1", PlateNumber: "P2"})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("期望冲突错误，得到: %v", err)
	}
}

func TestVehicleService_UpdateMileage_RejectsDecrease(t *testing.T) {
	vehRepo := newFakeVehicleRepo()
	tx := &fakeTransactor{stores: repository.Stores{Vehicles: vehRepo}}
	auditSvc := &AuditService{repo: &nopAuditRepo{}}
	svc := NewVehicleService(vehRepo, tx, auditSvc, nil)

	v, _ := svc.Create(context.Background(), entity.Vehicle{Model: "A", VIN: "V1", PlateNumber: "P1"})
	_ = svc.UpdateMileage(context.Background(), v.ID, 1000)
	if err := svc.UpdateMileage(context.Background(), v.ID, 900); !errors.Is(err, domain.ErrMileageNotIncreasing) {
		t.Fatalf("期望里程递减错误，得到: %v", err)
	}
}

func TestVehicleService_ChangeStatus_RespectsLegalTransition(t *testing.T) {
	vehRepo := newFakeVehicleRepo()
	tx := &fakeTransactor{stores: repository.Stores{Vehicles: vehRepo}}
	auditSvc := &AuditService{repo: &nopAuditRepo{}}
	svc := NewVehicleService(vehRepo, tx, auditSvc, nil)

	v, _ := svc.Create(context.Background(), entity.Vehicle{Model: "A", VIN: "V1", PlateNumber: "P1"})
	// active -> in_maintenance 合法。
	if err := svc.ChangeStatus(context.Background(), v.ID, entity.VehicleStatusInMaintenance, "保养", entity.AuditActor{}); err != nil {
		t.Fatalf("合法流转失败: %v", err)
	}
	// in_maintenance -> retired 合法。
	if err := svc.ChangeStatus(context.Background(), v.ID, entity.VehicleStatusRetired, "报废", entity.AuditActor{}); err != nil {
		t.Fatalf("合法流转失败: %v", err)
	}
	// retired -> active 非法。
	if err := svc.ChangeStatus(context.Background(), v.ID, entity.VehicleStatusActive, "复活", entity.AuditActor{}); !errors.Is(err, domain.ErrStateTransition) {
		t.Fatalf("期望状态流转错误，得到: %v", err)
	}
}

func TestCanTransitionVehicle(t *testing.T) {
	cases := []struct {
		from, to string
		want     bool
	}{
		{entity.VehicleStatusActive, entity.VehicleStatusInMaintenance, true},
		{entity.VehicleStatusInMaintenance, entity.VehicleStatusActive, true},
		{entity.VehicleStatusRetired, entity.VehicleStatusActive, false},
		{entity.VehicleStatusActive, entity.VehicleStatusActive, true},
	}
	for _, c := range cases {
		if got := entity.CanTransitionVehicle(c.from, c.to); got != c.want {
			t.Fatalf("CanTransitionVehicle(%s,%s)=%v want %v", c.from, c.to, got, c.want)
		}
	}
}

func TestCanTransitionTrip(t *testing.T) {
	cases := []struct {
		from, to string
		want     bool
	}{
		{entity.TripStatusScheduled, entity.TripStatusInProgress, true},
		{entity.TripStatusInProgress, entity.TripStatusCompleted, true},
		{entity.TripStatusCompleted, entity.TripStatusCancelled, false},
		{entity.TripStatusCancelled, entity.TripStatusInProgress, false},
	}
	for _, c := range cases {
		if got := entity.CanTransitionTrip(c.from, c.to); got != c.want {
			t.Fatalf("CanTransitionTrip(%s,%s)=%v want %v", c.from, c.to, got, c.want)
		}
	}
}

// nopAuditRepo 审计仓储空实现。
type nopAuditRepo struct{}

func (n *nopAuditRepo) Append(_ context.Context, a entity.AuditLog) (entity.AuditLog, error) {
	return a, nil
}
func (n *nopAuditRepo) List(_ context.Context, page entity.Page, filter entity.Filter) ([]entity.AuditLog, int64, error) {
	return nil, 0, nil
}
