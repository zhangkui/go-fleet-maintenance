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

// fakeTripRepo 最小实现，供 trip service 测试。
type fakeTripRepo struct {
	trips map[int64]entity.Trip
	next  int64
}

func newFakeTripRepo() *fakeTripRepo { return &fakeTripRepo{trips: map[int64]entity.Trip{}} }

func (r *fakeTripRepo) CreateTrip(_ context.Context, t entity.Trip) (entity.Trip, error) {
	r.next++
	t.ID = r.next
	r.trips[t.ID] = t
	return t, nil
}
func (r *fakeTripRepo) GetTripByID(_ context.Context, id int64) (entity.Trip, error) {
	t, ok := r.trips[id]
	if !ok {
		return entity.Trip{}, domain.ErrNotFound
	}
	return t, nil
}
func (r *fakeTripRepo) GetTripByIDForUpdate(_ context.Context, id int64) (entity.Trip, error) {
	return r.GetTripByID(context.Background(), id)
}
func (r *fakeTripRepo) ListTrips(_ context.Context, page entity.Page, filter entity.Filter, sort entity.Sort) ([]entity.Trip, int64, error) {
	out := make([]entity.Trip, 0, len(r.trips))
	for _, t := range r.trips {
		out = append(out, t)
	}
	return out, int64(len(out)), nil
}
func (r *fakeTripRepo) StartTrip(_ context.Context, id int64, by int64, at time.Time) error {
	t := r.trips[id]
	t.Status = entity.TripStatusInProgress
	r.trips[id] = t
	return nil
}
func (r *fakeTripRepo) CompleteTrip(_ context.Context, id int64, endOdometer int64, completedBy int64, at time.Time) error {
	t := r.trips[id]
	t.Status = entity.TripStatusCompleted
	ed := endOdometer
	t.EndOdometerKM = &ed
	r.trips[id] = t
	return nil
}
func (r *fakeTripRepo) CancelTrip(_ context.Context, id int64, by int64, at time.Time) error {
	t := r.trips[id]
	t.Status = entity.TripStatusCancelled
	r.trips[id] = t
	return nil
}
func (r *fakeTripRepo) AppendTripStatusHistory(_ context.Context, h entity.TripStatusHistory) error {
	return nil
}
func (r *fakeTripRepo) CreateHandover(_ context.Context, h entity.TripHandover) (entity.TripHandover, error) {
	return h, nil
}

func TestFuelService_Record_RejectsMileageDecrease(t *testing.T) {
	vehRepo := newFakeVehicleRepo()
	fuelRepo := newFakeFuelRepo()
	stores := repository.Stores{Vehicles: vehRepo, Fuel: fuelRepo}
	tx := &fakeTransactor{stores: stores}
	auditSvc := &AuditService{repo: &nopAuditRepo{}}
	svc := NewFuelService(fuelRepo, vehRepo, tx, auditSvc, nil)

	v, _ := (&VehicleService{repo: vehRepo, tx: tx, audit: auditSvc}).Create(context.Background(), entity.Vehicle{Model: "A", VIN: "V1", PlateNumber: "P1"})
	_, err := svc.Record(context.Background(), entity.FuelInput{VehicleID: v.ID, LitersMilli: 40000, UnitPriceCents: 800, OdometerKM: 500, IdempotencyKey: "k1"}, entity.AuditActor{})
	if err != nil {
		t.Fatalf("首次录入失败: %v", err)
	}
	// 里程递减应被拒绝。
	_, err = svc.Record(context.Background(), entity.FuelInput{VehicleID: v.ID, LitersMilli: 30000, UnitPriceCents: 800, OdometerKM: 400, IdempotencyKey: "k2"}, entity.AuditActor{})
	if !errors.Is(err, domain.ErrMileageNotIncreasing) {
		t.Fatalf("期望里程递减错误，得到: %v", err)
	}
}

func TestFuelService_Record_ComputesTotalCost(t *testing.T) {
	vehRepo := newFakeVehicleRepo()
	fuelRepo := newFakeFuelRepo()
	stores := repository.Stores{Vehicles: vehRepo, Fuel: fuelRepo}
	tx := &fakeTransactor{stores: stores}
	auditSvc := &AuditService{repo: &nopAuditRepo{}}
	svc := NewFuelService(fuelRepo, vehRepo, tx, auditSvc, nil)

	v, _ := (&VehicleService{repo: vehRepo, tx: tx, audit: auditSvc}).Create(context.Background(), entity.Vehicle{Model: "A", VIN: "V2", PlateNumber: "P2"})
	// 40 升（40000 毫升）* 8 元/升（800 分）= 320 元 = 32000 分；total = 40000*800/1000 = 32000。
	rec, err := svc.Record(context.Background(), entity.FuelInput{VehicleID: v.ID, LitersMilli: 40000, UnitPriceCents: 800, OdometerKM: 100, IdempotencyKey: "total1"}, entity.AuditActor{})
	if err != nil {
		t.Fatalf("录入失败: %v", err)
	}
	if rec.TotalCostCents != 32000 {
		t.Fatalf("总价计算错误: 得到 %d 期望 32000", rec.TotalCostCents)
	}
}

func TestFuelService_Record_RejectsDuplicateIdempotency(t *testing.T) {
	vehRepo := newFakeVehicleRepo()
	fuelRepo := newFakeFuelRepo()
	stores := repository.Stores{Vehicles: vehRepo, Fuel: fuelRepo}
	tx := &fakeTransactor{stores: stores}
	auditSvc := &AuditService{repo: &nopAuditRepo{}}
	svc := NewFuelService(fuelRepo, vehRepo, tx, auditSvc, nil)

	v, _ := (&VehicleService{repo: vehRepo, tx: tx, audit: auditSvc}).Create(context.Background(), entity.Vehicle{Model: "A", VIN: "V3", PlateNumber: "P3"})
	in := entity.FuelInput{VehicleID: v.ID, LitersMilli: 20000, UnitPriceCents: 700, OdometerKM: 200, IdempotencyKey: "dup1"}
	if _, err := svc.Record(context.Background(), in, entity.AuditActor{}); err != nil {
		t.Fatalf("首次录入失败: %v", err)
	}
	// 同幂等键第二次应被拒绝。
	if _, err := svc.Record(context.Background(), in, entity.AuditActor{}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("期望冲突错误，得到: %v", err)
	}
}
