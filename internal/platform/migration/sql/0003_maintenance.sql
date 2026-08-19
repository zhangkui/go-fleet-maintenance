-- 0003_maintenance.sql: 维保计划、工单、配件、库存流水
CREATE TABLE IF NOT EXISTS maintenance_policies (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    vehicle_id BIGINT NOT NULL,
    name VARCHAR(120) NOT NULL,
    kind VARCHAR(32) NOT NULL DEFAULT 'periodic',
    interval_km BIGINT NOT NULL DEFAULT 0,
    interval_days INT NOT NULL DEFAULT 0,
    last_service_km BIGINT NOT NULL DEFAULT 0,
    last_service_at DATE NOT NULL,
    next_due_km BIGINT NOT NULL DEFAULT 0,
    next_due_at DATE NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (vehicle_id) REFERENCES vehicles(id) ON DELETE CASCADE,
    KEY idx_policy_vehicle_due(vehicle_id, enabled, next_due_km, next_due_at),
    KEY idx_policy_due(next_due_km, next_due_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS maintenance_orders (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    vehicle_id BIGINT NOT NULL,
    policy_id BIGINT NULL,
    kind VARCHAR(32) NOT NULL,
    title VARCHAR(255) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    idempotency_key VARCHAR(64) NOT NULL DEFAULT '',
    downtime_start TIMESTAMP NULL,
    downtime_end TIMESTAMP NULL,
    parts_cost_cents BIGINT NOT NULL DEFAULT 0,
    labor_cost_cents BIGINT NOT NULL DEFAULT 0,
    total_cost_cents BIGINT NOT NULL DEFAULT 0,
    completed_at TIMESTAMP NULL,
    created_by BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (vehicle_id) REFERENCES vehicles(id),
    FOREIGN KEY (policy_id) REFERENCES maintenance_policies(id) ON DELETE SET NULL,
    UNIQUE KEY uq_order_idem(idempotency_key),
    KEY idx_order_vehicle_status(vehicle_id, status),
    KEY idx_order_status(status, created_at),
    KEY idx_order_policy(policy_id, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- parts 表先于 maintenance_order_parts 外键被引用，放前面（CREATE IF NOT EXISTS 幂等）。
CREATE TABLE IF NOT EXISTS parts (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    sku VARCHAR(64) NOT NULL UNIQUE,
    name VARCHAR(120) NOT NULL,
    unit VARCHAR(16) NOT NULL DEFAULT '件',
    stock_quantity BIGINT NOT NULL DEFAULT 0,
    reorder_point BIGINT NOT NULL DEFAULT 0,
    unit_cost_cents BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY idx_part_stock(stock_quantity),
    KEY idx_part_reorder(reorder_point)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS maintenance_order_parts (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    order_id BIGINT NOT NULL,
    part_id BIGINT NOT NULL,
    quantity BIGINT NOT NULL,
    unit_cost_cents BIGINT NOT NULL,
    line_total_cents BIGINT NOT NULL,
    FOREIGN KEY (order_id) REFERENCES maintenance_orders(id) ON DELETE CASCADE,
    FOREIGN KEY (part_id) REFERENCES parts(id),
    KEY idx_mop_order(order_id),
    KEY idx_mop_part(part_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS maintenance_order_status_history (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    order_id BIGINT NOT NULL,
    from_status VARCHAR(32) NOT NULL,
    to_status VARCHAR(32) NOT NULL,
    note VARCHAR(255) NOT NULL DEFAULT '',
    changed_by BIGINT NOT NULL DEFAULT 0,
    changed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (order_id) REFERENCES maintenance_orders(id) ON DELETE CASCADE,
    KEY idx_mosh_order(order_id, changed_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS part_stock_movements (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    part_id BIGINT NOT NULL,
    change_quantity BIGINT NOT NULL,
    reason VARCHAR(32) NOT NULL,
    ref_order_id BIGINT NULL,
    balance_after BIGINT NOT NULL,
    created_by BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (part_id) REFERENCES parts(id) ON DELETE CASCADE,
    KEY idx_psm_part(part_id, created_at),
    KEY idx_psm_reason(reason, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
