-- 0002_fleet.sql: 车辆、司机、出车任务、油耗
CREATE TABLE IF NOT EXISTS vehicles (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    model VARCHAR(120) NOT NULL,
    vin VARCHAR(32) NOT NULL UNIQUE,
    plate_number VARCHAR(32) NOT NULL UNIQUE,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    odometer_km BIGINT NOT NULL DEFAULT 0,
    color VARCHAR(32) NOT NULL DEFAULT '',
    engine_no VARCHAR(64) NOT NULL DEFAULT '',
    purchase_date DATE NULL,
    insurance_expiry DATE NULL,
    inspection_expiry DATE NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY idx_vehicle_status(status),
    KEY idx_vehicle_timeline(id, odometer_km, updated_at),
    KEY idx_vehicle_insurance(insurance_expiry),
    KEY idx_vehicle_inspection(inspection_expiry)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS vehicle_status_history (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    vehicle_id BIGINT NOT NULL,
    from_status VARCHAR(32) NOT NULL,
    to_status VARCHAR(32) NOT NULL,
    reason VARCHAR(255) NOT NULL DEFAULT '',
    changed_by BIGINT NOT NULL DEFAULT 0,
    changed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (vehicle_id) REFERENCES vehicles(id) ON DELETE CASCADE,
    KEY idx_vsh_vehicle(vehicle_id, changed_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS vehicle_licenses (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    vehicle_id BIGINT NOT NULL,
    kind VARCHAR(32) NOT NULL,
    number VARCHAR(96) NOT NULL DEFAULT '',
    issue_date DATE NULL,
    expiry_date DATE NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (vehicle_id) REFERENCES vehicles(id) ON DELETE CASCADE,
    KEY idx_vl_vehicle(vehicle_id),
    KEY idx_vl_expiry(kind, expiry_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS drivers (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(120) NOT NULL,
    license_number VARCHAR(64) NOT NULL UNIQUE,
    license_class VARCHAR(16) NOT NULL,
    license_expiry DATE NOT NULL,
    phone VARCHAR(32) NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY idx_driver_status(status),
    KEY idx_driver_license_expiry(license_expiry)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS driver_vehicle_bindings (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    driver_id BIGINT NOT NULL,
    vehicle_id BIGINT NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (driver_id) REFERENCES drivers(id) ON DELETE CASCADE,
    FOREIGN KEY (vehicle_id) REFERENCES vehicles(id) ON DELETE CASCADE,
    KEY idx_binding_vehicle(vehicle_id, status, start_date, end_date),
    KEY idx_binding_driver(driver_id, status, start_date, end_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS driver_schedules (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    driver_id BIGINT NOT NULL,
    shift_date DATE NOT NULL,
    shift_type VARCHAR(16) NOT NULL,
    note VARCHAR(255) NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (driver_id) REFERENCES drivers(id) ON DELETE CASCADE,
    KEY idx_schedule_driver_date(driver_id, shift_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS driver_violations (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    driver_id BIGINT NOT NULL,
    occurred_at TIMESTAMP NOT NULL,
    violation_type VARCHAR(64) NOT NULL,
    points INT NOT NULL DEFAULT 0,
    fine_cents BIGINT NOT NULL DEFAULT 0,
    description VARCHAR(255) NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (driver_id) REFERENCES drivers(id) ON DELETE CASCADE,
    KEY idx_violation_driver(driver_id, occurred_at),
    KEY idx_violation_status(status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS trips (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    vehicle_id BIGINT NOT NULL,
    driver_id BIGINT NOT NULL,
    route VARCHAR(500) NOT NULL,
    load_kg BIGINT NOT NULL DEFAULT 0,
    start_odometer_km BIGINT NOT NULL DEFAULT 0,
    end_odometer_km BIGINT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'scheduled',
    idempotency_key VARCHAR(64) NOT NULL DEFAULT '',
    completed_at TIMESTAMP NULL,
    completed_by BIGINT NULL,
    note VARCHAR(500) NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (vehicle_id) REFERENCES vehicles(id),
    FOREIGN KEY (driver_id) REFERENCES drivers(id),
    UNIQUE KEY uq_trip_idem(idempotency_key),
    KEY idx_trip_vehicle_timeline(vehicle_id, status, completed_at),
    KEY idx_trip_driver(driver_id, status),
    KEY idx_trip_status(status, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS trip_status_history (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    trip_id BIGINT NOT NULL,
    from_status VARCHAR(32) NOT NULL,
    to_status VARCHAR(32) NOT NULL,
    note VARCHAR(255) NOT NULL DEFAULT '',
    changed_by BIGINT NOT NULL DEFAULT 0,
    changed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (trip_id) REFERENCES trips(id) ON DELETE CASCADE,
    KEY idx_tsh_trip(trip_id, changed_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS trip_handovers (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    trip_id BIGINT NOT NULL,
    from_driver_id BIGINT NOT NULL,
    to_driver_id BIGINT NOT NULL,
    handover_at TIMESTAMP NOT NULL,
    location VARCHAR(255) NOT NULL DEFAULT '',
    note VARCHAR(255) NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (trip_id) REFERENCES trips(id) ON DELETE CASCADE,
    FOREIGN KEY (from_driver_id) REFERENCES drivers(id),
    FOREIGN KEY (to_driver_id) REFERENCES drivers(id),
    KEY idx_handover_trip(trip_id, handover_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS fuel_records (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    vehicle_id BIGINT NOT NULL,
    liters_milli BIGINT NOT NULL,
    unit_price_cents BIGINT NOT NULL,
    odometer_km BIGINT NOT NULL,
    total_cost_cents BIGINT NOT NULL,
    abnormal BOOLEAN NOT NULL DEFAULT FALSE,
    recorded_at TIMESTAMP NOT NULL,
    idempotency_key VARCHAR(64) NOT NULL DEFAULT '',
    created_by BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (vehicle_id) REFERENCES vehicles(id),
    UNIQUE KEY uq_fuel_idem(idempotency_key),
    KEY idx_fuel_vehicle_timeline(vehicle_id, odometer_km, recorded_at),
    KEY idx_fuel_abnormal(abnormal),
    KEY idx_fuel_recorded(recorded_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
