package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zhangkui/go-fleet-maintenance/internal/domain"
	"github.com/zhangkui/go-fleet-maintenance/internal/domain/entity"
	"github.com/zhangkui/go-fleet-maintenance/internal/repository"
)

func TestTripService_Complete_EnforcesMileageMonotonic(t *testing.T) {
	vehRepo := newFakeVehicleRepo()
	tripRepo := newFakeTripRepo()
	stores := repository.Stores{Vehicles: vehRepo, Trips: tripRepo}
	tx := &fakeTransactor{stores: stores}
	auditSvc := &AuditService{repo: &nopAuditRepo{}}
	// 构造一个里程已为 2000 的车辆。
	v, _ := vehRepo.CreateVehicle(context.Background(), entity.Vehicle{Model: "A", VIN: "VT1", PlateNumber: "PT1", OdometerKM: 2000, Status: entity.VehicleStatusActive})

	svc := NewTripService(tripRepo, vehRepo, nil, nil, tx, auditSvc, nil)
	// 创建任务，起点里程 2000。
	tr, err := svc.Create(context.Background(), entity.Trip{VehicleID: v.ID, DriverID: 1, Route: "r", StartOdometerKM: 2000, Status: entity.TripStatusScheduled, IdempotencyKey: "t1"}, entity.AuditActor{UserID: 1})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	// 先开始任务。
	if err := svc.Start(context.Background(), tr.ID, entity.AuditActor{UserID: 1}); err != nil {
		t.Fatalf("start: %v", err)
	}
	// 结束里程低于车辆里程应被拒绝。
	_, err = svc.Complete(context.Background(), tr.ID, entity.TripComplete{EndOdometerKM: 1500, CompletedAt: time.Now()}, entity.AuditActor{UserID: 1})
	if !errors.Is(err, domain.ErrMileageNotIncreasing) {
		t.Fatalf("期望里程递减错误，得到: %v", err)
	}
	// 合法完成。
	_, err = svc.Complete(context.Background(), tr.ID, entity.TripComplete{EndOdometerKM: 3000, CompletedAt: time.Now()}, entity.AuditActor{UserID: 1})
	if err != nil {
		t.Fatalf("合法完成失败: %v", err)
	}
	// 车辆里程应被更新到 3000。
	v2, _ := vehRepo.GetVehicleByID(context.Background(), v.ID)
	if v2.OdometerKM != 3000 {
		t.Fatalf("车辆里程未更新: %d", v2.OdometerKM)
	}
}

func TestTripService_Complete_RejectsIllegalTransition(t *testing.T) {
	vehRepo := newFakeVehicleRepo()
	tripRepo := newFakeTripRepo()
	stores := repository.Stores{Vehicles: vehRepo, Trips: tripRepo}
	tx := &fakeTransactor{stores: stores}
	auditSvc := &AuditService{repo: &nopAuditRepo{}}
	v, _ := vehRepo.CreateVehicle(context.Background(), entity.Vehicle{Model: "A", VIN: "VT2", PlateNumber: "PT2", Status: entity.VehicleStatusActive})

	svc := NewTripService(tripRepo, vehRepo, nil, nil, tx, auditSvc, nil)
	tr, _ := svc.Create(context.Background(), entity.Trip{VehicleID: v.ID, DriverID: 1, Route: "r", StartOdometerKM: 0, IdempotencyKey: "t2"}, entity.AuditActor{UserID: 1})
	// 未经 start 直接 complete 是非法的（scheduled -> completed 不在流转图）。
	_, err := svc.Complete(context.Background(), tr.ID, entity.TripComplete{EndOdometerKM: 100, CompletedAt: time.Now()}, entity.AuditActor{UserID: 1})
	if !errors.Is(err, domain.ErrStateTransition) {
		t.Fatalf("期望状态流转错误，得到: %v", err)
	}
}
