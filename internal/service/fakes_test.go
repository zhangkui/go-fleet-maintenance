package service

import (
	"context"
	"sync"
	"time"

	"github.com/zhangkui/go-fleet-maintenance/internal/domain"
	"github.com/zhangkui/go-fleet-maintenance/internal/domain/entity"
)

// fakeVehicleRepo 内存实现 VehicleRepository，用于隔离 MySQL 的单元测试。
type fakeVehicleRepo struct {
	mu       sync.Mutex
	byID     map[int64]entity.Vehicle
	nextID   int64
	history  []entity.VehicleStatusHistory
	licenses []entity.VehicleLicense
	expiring []entity.Vehicle
}

func newFakeVehicleRepo() *fakeVehicleRepo { return &fakeVehicleRepo{byID: map[int64]entity.Vehicle{}} }

func (r *fakeVehicleRepo) CreateVehicle(ctx context.Context, v entity.Vehicle) (entity.Vehicle, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, e := range r.byID {
		if e.VIN == v.VIN || e.PlateNumber == v.PlateNumber {
			return entity.Vehicle{}, domain.ErrConflict
		}
	}
	r.nextID++
	v.ID = r.nextID
	v.CreatedAt = time.Now()
	v.UpdatedAt = v.CreatedAt
	r.byID[v.ID] = v
	return v, nil
}
func (r *fakeVehicleRepo) GetVehicleByID(ctx context.Context, id int64) (entity.Vehicle, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.byID[id]
	if !ok {
		return entity.Vehicle{}, domain.ErrNotFound
	}
	return v, nil
}
func (r *fakeVehicleRepo) GetVehicleByIDForUpdate(ctx context.Context, id int64) (entity.Vehicle, error) {
	return r.GetVehicleByID(ctx, id)
}
func (r *fakeVehicleRepo) ListVehicles(ctx context.Context, page entity.Page, filter entity.Filter, sort entity.Sort) ([]entity.Vehicle, int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]entity.Vehicle, 0, len(r.byID))
	for _, v := range r.byID {
		out = append(out, v)
	}
	return out, int64(len(out)), nil
}
func (r *fakeVehicleRepo) UpdateVehicleStatus(ctx context.Context, id int64, status string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.byID[id]
	if !ok {
		return domain.ErrNotFound
	}
	v.Status = status
	r.byID[id] = v
	return nil
}
func (r *fakeVehicleRepo) UpdateVehicleMileage(ctx context.Context, id int64, odometer int64, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.byID[id]
	if !ok {
		return domain.ErrNotFound
	}
	v.OdometerKM = odometer
	r.byID[id] = v
	return nil
}
func (r *fakeVehicleRepo) UpdateVehicleLicenseExpiry(ctx context.Context, id int64, kind string, expiry time.Time) error {
	return nil
}
func (r *fakeVehicleRepo) AppendVehicleStatusHistory(ctx context.Context, h entity.VehicleStatusHistory) error {
	r.history = append(r.history, h)
	return nil
}
func (r *fakeVehicleRepo) ListVehicleStatusHistory(ctx context.Context, vehicleID int64) ([]entity.VehicleStatusHistory, error) {
	out := []entity.VehicleStatusHistory{}
	for _, h := range r.history {
		if h.VehicleID == vehicleID {
			out = append(out, h)
		}
	}
	return out, nil
}
func (r *fakeVehicleRepo) CreateVehicleLicense(ctx context.Context, l entity.VehicleLicense) (entity.VehicleLicense, error) {
	l.ID = int64(len(r.licenses) + 1)
	l.CreatedAt = time.Now()
	r.licenses = append(r.licenses, l)
	return l, nil
}
func (r *fakeVehicleRepo) ListVehicleLicenses(ctx context.Context, vehicleID int64) ([]entity.VehicleLicense, error) {
	out := []entity.VehicleLicense{}
	for _, l := range r.licenses {
		if l.VehicleID == vehicleID {
			out = append(out, l)
		}
	}
	return out, nil
}
func (r *fakeVehicleRepo) ListExpiringDocuments(ctx context.Context, from, to time.Time) ([]entity.Vehicle, error) {
	return r.expiring, nil
}

// fakeFuelRepo 内存实现 FuelRepository。
type fakeFuelRepo struct {
	mu   sync.Mutex
	recs []entity.FuelRecord
	next int64
	idem map[string]bool
}

func newFakeFuelRepo() *fakeFuelRepo { return &fakeFuelRepo{idem: map[string]bool{}} }

func (r *fakeFuelRepo) CreateFuelRecord(ctx context.Context, f entity.FuelRecord) (entity.FuelRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.idem[f.IdempotencyKey] {
		return entity.FuelRecord{}, domain.ErrConflict
	}
	r.idem[f.IdempotencyKey] = true
	r.next++
	f.ID = r.next
	f.CreatedAt = time.Now()
	r.recs = append(r.recs, f)
	return f, nil
}
func (r *fakeFuelRepo) GetFuelRecord(ctx context.Context, id int64) (entity.FuelRecord, error) {
	for _, f := range r.recs {
		if f.ID == id {
			return f, nil
		}
	}
	return entity.FuelRecord{}, domain.ErrNotFound
}
func (r *fakeFuelRepo) ListFuelRecords(ctx context.Context, page entity.Page, filter entity.Filter, sort entity.Sort) ([]entity.FuelRecord, int64, error) {
	return r.recs, int64(len(r.recs)), nil
}
func (r *fakeFuelRepo) LastFuelRecord(ctx context.Context, vehicleID int64) (entity.FuelRecord, error) {
	var last entity.FuelRecord
	found := false
	for _, f := range r.recs {
		if f.VehicleID == vehicleID && (!found || f.OdometerKM > last.OdometerKM) {
			last = f
			found = true
		}
	}
	if !found {
		return entity.FuelRecord{}, domain.ErrNotFound
	}
	return last, nil
}
func (r *fakeFuelRepo) RecentFuelRecords(ctx context.Context, vehicleID int64, limit int) ([]entity.FuelRecord, error) {
	var matched []entity.FuelRecord
	for _, f := range r.recs {
		if f.VehicleID == vehicleID {
			matched = append(matched, f)
		}
	}
	// 按里程升序
	for i := 0; i < len(matched); i++ {
		for j := i + 1; j < len(matched); j++ {
			if matched[j].OdometerKM < matched[i].OdometerKM {
				matched[i], matched[j] = matched[j], matched[i]
			}
		}
	}
	if len(matched) > limit {
		matched = matched[len(matched)-limit:]
	}
	return matched, nil
}
