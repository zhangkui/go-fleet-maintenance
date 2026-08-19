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

func TestBug007_VehicleMileageTimestamps(t *testing.T) {
	now := time.Date(2026, 8, 19, 10, 0, 0, 0, time.UTC)
	repo := &vehicleStub{vehicle: entity.Vehicle{ID: 1, OdometerKM: 100}}
	svc := service.NewVehicleService(repo, nil, nil, nil)
	if err := svc.UpdateMileage(context.Background(), 1, 100); err != nil {
		t.Fatalf("equal mileage: %v", err)
	}
	before := time.Now()
	if err := svc.UpdateMileage(context.Background(), 1, 120); err != nil {
		t.Fatal(err)
	}
	if repo.updatedAt.Before(before) {
		t.Fatalf("updated_at=%v", repo.updatedAt)
	}
	db, mock, _ := sqlmock.New()
	defer db.Close()
	mock.ExpectExec("UPDATE vehicles SET odometer_km=\\?, updated_at=\\? WHERE id=\\?").WithArgs(int64(120), sqlmock.AnyArg(), int64(1)).WillReturnResult(sqlmock.NewResult(0, 1))
	if err := mysqlrepo.NewVehicleRepository(db).UpdateVehicleMileage(context.Background(), 1, 120, now); err != nil {
		t.Fatal(err)
	}
}

func TestBug008_VehicleStatusHistory(t *testing.T) {
	repo := &vehicleStub{vehicle: entity.Vehicle{ID: 1, Status: entity.VehicleStatusActive}}
	svc := service.NewVehicleService(repo, transactorStub{stores: repository.Stores{Vehicles: repo}}, nil, nil)
	if err := svc.ChangeStatus(context.Background(), 1, entity.VehicleStatusInMaintenance, "service", entity.AuditActor{UserID: 9}); err != nil {
		t.Fatal(err)
	}
	if repo.history.FromStatus != entity.VehicleStatusActive || repo.history.ToStatus != entity.VehicleStatusInMaintenance {
		t.Fatalf("history=%+v", repo.history)
	}
	db, mock, _ := sqlmock.New()
	defer db.Close()
	mock.ExpectExec("INSERT INTO vehicle_status_history").WithArgs(int64(1), entity.VehicleStatusActive, entity.VehicleStatusInMaintenance, "service", int64(9)).WillReturnResult(sqlmock.NewResult(1, 1))
	if err := mysqlrepo.NewVehicleRepository(db).AppendVehicleStatusHistory(context.Background(), repo.history); err != nil {
		t.Fatal(err)
	}
}

func TestBug009_VehicleLicenseExpiry(t *testing.T) {
	expiry := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	repo := &vehicleStub{}
	svc := service.NewVehicleService(repo, nil, nil, nil)
	if _, err := svc.AddLicense(context.Background(), entity.VehicleLicense{VehicleID: 1, Kind: entity.LicenseKindInsurance, Number: "POLICY-1", ExpiryDate: &expiry}); err != nil {
		t.Fatal(err)
	}
	if repo.expiryKind != entity.LicenseKindInsurance {
		t.Fatalf("kind=%q", repo.expiryKind)
	}
	db, mock, _ := sqlmock.New()
	defer db.Close()
	mock.ExpectExec("UPDATE vehicles SET insurance_expiry=\\? WHERE id=\\?").WithArgs(expiry, int64(1)).WillReturnResult(sqlmock.NewResult(0, 1))
	if err := mysqlrepo.NewVehicleRepository(db).UpdateVehicleLicenseExpiry(context.Background(), 1, entity.LicenseKindInsurance, expiry); err != nil {
		t.Fatal(err)
	}
}

func TestBug010_DriverVehicleBindingWindow(t *testing.T) {
	start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	repo := &driverStub{}
	svc := service.NewDriverService(repo, nil)
	if _, err := svc.CreateBinding(context.Background(), entity.DriverVehicleBinding{DriverID: 2, VehicleID: 3, StartDate: start}); err != nil {
		t.Fatal(err)
	}
	if !repo.checkedAt.Equal(start) {
		t.Fatalf("checked at=%v want=%v", repo.checkedAt, start)
	}
	db, mock, _ := sqlmock.New()
	defer db.Close()
	mock.ExpectQuery("status=\\? AND start_date<=\\? AND \\(end_date IS NULL OR end_date>=\\?\\)").WithArgs(int64(3), entity.BindingStatusActive, "2026-09-01", "2026-09-01").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	if _, err := mysqlrepo.NewDriverRepository(db).HasActiveBindingForVehicle(context.Background(), 3, start); err != nil {
		t.Fatal(err)
	}
}
