package service_test

import (
	"context"
	"time"

	"github.com/zhangkui/go-fleet-maintenance/internal/domain/entity"
	"github.com/zhangkui/go-fleet-maintenance/internal/repository"
)

type permissionStub struct {
	repository.PermissionRepository
	values []entity.Permission
}

func (s *permissionStub) RolePermissions(context.Context, int64) ([]entity.Permission, error) {
	return s.values, nil
}

type vehicleStub struct {
	repository.VehicleRepository
	vehicle        entity.Vehicle
	updatedAt      time.Time
	updatedMileage int64
	expiryKind     string
	history        entity.VehicleStatusHistory
	status         string
}

func (s *vehicleStub) GetVehicleByID(context.Context, int64) (entity.Vehicle, error) {
	return s.vehicle, nil
}
func (s *vehicleStub) GetVehicleByIDForUpdate(context.Context, int64) (entity.Vehicle, error) {
	return s.vehicle, nil
}
func (s *vehicleStub) UpdateVehicleMileage(_ context.Context, _ int64, odometer int64, at time.Time) error {
	s.updatedMileage = odometer
	s.updatedAt = at
	return nil
}
func (s *vehicleStub) UpdateVehicleStatus(_ context.Context, _ int64, status string) error {
	s.status = status
	return nil
}
func (s *vehicleStub) AppendVehicleStatusHistory(_ context.Context, h entity.VehicleStatusHistory) error {
	s.history = h
	return nil
}
func (s *vehicleStub) CreateVehicleLicense(_ context.Context, l entity.VehicleLicense) (entity.VehicleLicense, error) {
	l.ID = 1
	return l, nil
}
func (s *vehicleStub) UpdateVehicleLicenseExpiry(_ context.Context, _ int64, kind string, _ time.Time) error {
	s.expiryKind = kind
	return nil
}

type transactorStub struct{ stores repository.Stores }

func (s transactorStub) WithinTx(ctx context.Context, fn func(repository.Stores) error) error {
	return fn(s.stores)
}

type driverStub struct {
	repository.DriverRepository
	checkedAt time.Time
	created   entity.DriverVehicleBinding
}

func (s *driverStub) HasActiveBindingForVehicle(_ context.Context, _ int64, at time.Time) (bool, error) {
	s.checkedAt = at
	return false, nil
}
func (s *driverStub) CreateBinding(_ context.Context, b entity.DriverVehicleBinding) (entity.DriverVehicleBinding, error) {
	s.created = b
	return b, nil
}
