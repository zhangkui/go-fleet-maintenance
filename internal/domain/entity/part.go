package entity

import "time"

// Part 配件主实体。
type Part struct {
	ID            int64     `json:"id"`
	SKU           string    `json:"sku"`
	Name          string    `json:"name"`
	Unit          string    `json:"unit"`
	StockQuantity int64     `json:"stock_quantity"`
	ReorderPoint  int64     `json:"reorder_point"`
	UnitCostCents int64     `json:"unit_cost_cents"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// PartStockMovement 配件库存变动流水实体。
type PartStockMovement struct {
	ID             int64     `json:"id"`
	PartID         int64     `json:"part_id"`
	ChangeQuantity int64     `json:"change_quantity"`
	Reason         string    `json:"reason"`
	RefOrderID     *int64    `json:"ref_order_id,omitempty"`
	BalanceAfter   int64     `json:"balance_after"`
	CreatedAt      time.Time `json:"created_at"`
	CreatedBy      int64     `json:"created_by"`
}

// PartAdjust 配件库存调整请求。
type PartAdjust struct {
	PartID int64  `json:"part_id"`
	Change int64  `json:"change"`
	Reason string `json:"reason"`
	Note   string `json:"note"`
}
