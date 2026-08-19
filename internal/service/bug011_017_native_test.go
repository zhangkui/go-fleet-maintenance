package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/zhangkui/go-fleet-maintenance/internal/domain/entity"
	"github.com/zhangkui/go-fleet-maintenance/internal/repository"
	mysqlrepo "github.com/zhangkui/go-fleet-maintenance/internal/repository/mysql"
	"github.com/zhangkui/go-fleet-maintenance/internal/service"
)

type driverBehaviorStub struct {
	repository.DriverRepository
	schedule  entity.DriverSchedule
	violation entity.DriverViolation
}

func (s *driverBehaviorStub) CreateSchedule(_ context.Context, schedule entity.DriverSchedule) (entity.DriverSchedule, error) {
	s.schedule = schedule
	return schedule, nil
}

func (s *driverBehaviorStub) CreateViolation(_ context.Context, violation entity.DriverViolation) (entity.DriverViolation, error) {
	s.violation = violation
	violation.ID = 1
	return violation, nil
}

type auditCaptureStub struct {
	repository.AuditRepository
	entry entity.AuditLog
}

func (s *auditCaptureStub) Append(_ context.Context, entry entity.AuditLog) (entity.AuditLog, error) {
	s.entry = entry
	return entry, nil
}

type tripBehaviorStub struct {
	repository.TripRepository
	trip              entity.Trip
	completedOdometer int64
	startedAt         time.Time
	handover          entity.TripHandover
}

func (s *tripBehaviorStub) GetTripByIDForUpdate(context.Context, int64) (entity.Trip, error) {
	return s.trip, nil
}

func (s *tripBehaviorStub) GetTripByID(context.Context, int64) (entity.Trip, error) {
	return s.trip, nil
}

func (s *tripBehaviorStub) CompleteTrip(_ context.Context, _ int64, odometer int64, _ int64, _ time.Time) error {
	s.completedOdometer = odometer
	return nil
}

func (s *tripBehaviorStub) StartTrip(_ context.Context, _ int64, _ int64, at time.Time) error {
	s.startedAt = at
	return nil
}

func (s *tripBehaviorStub) AppendTripStatusHistory(context.Context, entity.TripStatusHistory) error {
	return nil
}

func (s *tripBehaviorStub) CreateHandover(_ context.Context, handover entity.TripHandover) (entity.TripHandover, error) {
	s.handover = handover
	return handover, nil
}

type fuelBehaviorStub struct {
	repository.FuelRepository
	recent  []entity.FuelRecord
	created entity.FuelRecord
}

func (s *fuelBehaviorStub) RecentFuelRecords(context.Context, int64, int) ([]entity.FuelRecord, error) {
	return s.recent, nil
}

func (s *fuelBehaviorStub) CreateFuelRecord(_ context.Context, record entity.FuelRecord) (entity.FuelRecord, error) {
	s.created = record
	record.ID = 1
	return record, nil
}

func TestBug011_DriverOffShiftValidation(t *testing.T) {
	repo := &driverBehaviorStub{}
	svc := service.NewDriverService(repo, nil)
	_, err := svc.CreateSchedule(context.Background(), entity.DriverSchedule{DriverID: 1, ShiftDate: time.Now(), ShiftType: "off"})
	if err != nil {
		t.Fatalf("off shift should be valid: %v", err)
	}
	if entity.ShiftOff != "off" {
		t.Fatalf("ShiftOff=%q", entity.ShiftOff)
	}
}

func TestBug012_DriverViolationFineCents(t *testing.T) {
	repo := &driverBehaviorStub{}
	auditRepo := &auditCaptureStub{}
	svc := service.NewDriverService(repo, service.NewAuditService(auditRepo))
	created, err := svc.CreateViolation(context.Background(), entity.DriverViolation{DriverID: 1, ViolationType: "speeding", FineCents: 12345}, entity.AuditActor{UserID: 2})
	if err != nil {
		t.Fatal(err)
	}
	if created.FineCents != 12345 || repo.violation.FineCents != 12345 {
		t.Fatalf("fine cents changed: created=%d stored=%d", created.FineCents, repo.violation.FineCents)
	}
}

func TestBug013_TripCompletionOdometer(t *testing.T) {
	tripRepo := &tripBehaviorStub{trip: entity.Trip{ID: 1, VehicleID: 2, Status: entity.TripStatusInProgress, StartOdometerKM: 100}}
	vehicleRepo := &vehicleStub{vehicle: entity.Vehicle{ID: 2, OdometerKM: 100}}
	tx := transactorStub{stores: repository.Stores{Trips: tripRepo, Vehicles: vehicleRepo}}
	svc := service.NewTripService(tripRepo, vehicleRepo, nil, nil, tx, service.NewAuditService(&auditCaptureStub{}), nil)
	_, err := svc.Complete(context.Background(), 1, entity.TripComplete{EndOdometerKM: 180, CompletedAt: time.Now()}, entity.AuditActor{UserID: 9})
	if err != nil {
		t.Fatal(err)
	}
	if tripRepo.completedOdometer != 180 || vehicleRepo.updatedMileage != 180 {
		t.Fatalf("trip=%d vehicle=%d", tripRepo.completedOdometer, vehicleRepo.updatedMileage)
	}

	db, mock, _ := sqlmock.New()
	defer db.Close()
	mock.ExpectExec("UPDATE trips SET status=\\?, end_odometer_km=\\?, completed_at=\\?, completed_by=\\?, updated_at=\\? WHERE id=\\?").
		WithArgs(entity.TripStatusCompleted, int64(180), sqlmock.AnyArg(), int64(9), sqlmock.AnyArg(), int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	if err := mysqlrepo.NewTripRepository(db).CompleteTrip(context.Background(), 1, 180, 9, time.Now()); err != nil {
		t.Fatal(err)
	}
}

func TestBug014_TripStartTimestamps(t *testing.T) {
	repo := &tripBehaviorStub{trip: entity.Trip{ID: 1, Status: entity.TripStatusScheduled}}
	tx := transactorStub{stores: repository.Stores{Trips: repo}}
	svc := service.NewTripService(repo, nil, nil, nil, tx, nil, nil)
	before := time.Now()
	if err := svc.Start(context.Background(), 1, entity.AuditActor{UserID: 5}); err != nil {
		t.Fatal(err)
	}
	if repo.startedAt.Before(before) {
		t.Fatalf("started_at=%v before=%v", repo.startedAt, before)
	}
	db, mock, _ := sqlmock.New()
	defer db.Close()
	mock.ExpectExec("UPDATE trips SET status=\\?, updated_at=\\? WHERE id=\\?").
		WithArgs(entity.TripStatusInProgress, sqlmock.AnyArg(), int64(1)).WillReturnResult(sqlmock.NewResult(0, 1))
	if err := mysqlrepo.NewTripRepository(db).StartTrip(context.Background(), 1, 5, time.Now()); err != nil {
		t.Fatal(err)
	}
}

func TestBug015_TripHandoverValidation(t *testing.T) {
	repo := &tripBehaviorStub{}
	svc := service.NewTripService(repo, nil, nil, nil, nil, nil, nil)
	invalid := []entity.TripHandover{
		{TripID: 1, FromDriverID: 0, ToDriverID: 2},
		{TripID: 1, FromDriverID: 2, ToDriverID: 0},
		{TripID: 1, FromDriverID: 2, ToDriverID: 2},
	}
	for _, handover := range invalid {
		if _, err := svc.CreateHandover(context.Background(), handover); err == nil {
			t.Fatalf("accepted invalid handover: %+v", handover)
		}
	}
}

func TestBug016_FuelCostRounding(t *testing.T) {
	fuelRepo := &fuelBehaviorStub{}
	vehicleRepo := &vehicleStub{vehicle: entity.Vehicle{ID: 1, OdometerKM: 100}}
	tx := transactorStub{stores: repository.Stores{Fuel: fuelRepo, Vehicles: vehicleRepo}}
	auditRepo := &auditCaptureStub{}
	svc := service.NewFuelService(fuelRepo, vehicleRepo, tx, service.NewAuditService(auditRepo), nil)
	created, err := svc.Record(context.Background(), entity.FuelInput{VehicleID: 1, LitersMilli: 1501, UnitPriceCents: 701, OdometerKM: 110, IdempotencyKey: "bug016"}, entity.AuditActor{UserID: 1})
	if err != nil {
		t.Fatal(err)
	}
	want := int64(1501 * 701 / 1000)
	if created.TotalCostCents != want || fuelRepo.created.TotalCostCents != want {
		t.Fatalf("cost=%d stored=%d want=%d", created.TotalCostCents, fuelRepo.created.TotalCostCents, want)
	}

	db, mock, _ := sqlmock.New()
	defer db.Close()
	mock.ExpectExec("INSERT INTO fuel_records").WithArgs(int64(1), int64(1501), int64(701), int64(110), want, false, sqlmock.AnyArg(), "bug016", int64(1)).WillReturnResult(sqlmock.NewResult(1, 1))
	if _, err := mysqlrepo.NewFuelRepository(db).CreateFuelRecord(context.Background(), entity.FuelRecord{VehicleID: 1, LitersMilli: 1501, UnitPriceCents: 701, OdometerKM: 110, TotalCostCents: want, IdempotencyKey: "bug016", CreatedBy: 1}); err != nil {
		t.Fatal(err)
	}
}

func TestBug017_FuelAnomalyDetection(t *testing.T) {
	fuelRepo := &fuelBehaviorStub{recent: []entity.FuelRecord{
		{OdometerKM: 100, LitersMilli: 0},
		{OdometerKM: 200, LitersMilli: 10000},
		{OdometerKM: 300, LitersMilli: 10000},
	}}
	vehicleRepo := &vehicleStub{vehicle: entity.Vehicle{ID: 1, OdometerKM: 300}}
	tx := transactorStub{stores: repository.Stores{Fuel: fuelRepo, Vehicles: vehicleRepo}}
	svc := service.NewFuelService(fuelRepo, vehicleRepo, tx, service.NewAuditService(&auditCaptureStub{}), nil)
	created, err := svc.Record(context.Background(), entity.FuelInput{VehicleID: 1, LitersMilli: 20000, UnitPriceCents: 700, OdometerKM: 400, IdempotencyKey: "bug017"}, entity.AuditActor{UserID: 1})
	if err != nil {
		t.Fatal(err)
	}
	if !created.Abnormal {
		t.Fatal("1.5x+ fuel consumption should be abnormal")
	}

	db, mock, _ := sqlmock.New()
	defer db.Close()
	rows := sqlmock.NewRows([]string{"id", "vehicle_id", "liters_milli", "unit_price_cents", "odometer_km", "total_cost_cents", "abnormal", "recorded_at", "idempotency_key", "created_by", "created_at"})
	mock.ExpectQuery("ORDER BY odometer_km DESC LIMIT \\?").WithArgs(int64(1), 5).WillReturnRows(rows)
	if _, err := mysqlrepo.NewFuelRepository(db).RecentFuelRecords(context.Background(), 1, 5); err != nil {
		t.Fatal(err)
	}
}
