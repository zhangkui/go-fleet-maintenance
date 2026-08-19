package mysqlrepo

import (
	"context"
	"database/sql"
	"time"

	"github.com/zhangkui/go-fleet-maintenance/internal/domain/entity"
)

// PartRepository 配件 MySQL 实现。
type PartRepository struct {
	db DBTX
}

// NewPartRepository 构造配件仓储。
func NewPartRepository(db DBTX) *PartRepository { return &PartRepository{db: db} }

// CreatePart 创建配件。
func (r *PartRepository) CreatePart(ctx context.Context, p entity.Part) (entity.Part, error) {
	res, err := r.db.ExecContext(ctx, `INSERT INTO parts(sku,name,unit,stock_quantity,reorder_point,unit_cost_cents)
		VALUES(?,?,?,?,?,?)`, p.SKU, p.Name, p.Unit, p.StockQuantity, p.ReorderPoint, p.UnitCostCents)
	if err != nil {
		return entity.Part{}, TranslateError(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return entity.Part{}, err
	}
	p.ID = id
	p.CreatedAt = time.Now()
	p.UpdatedAt = p.CreatedAt
	return p, nil
}

// scanPart 扫描配件行。
func scanPart(sc func(...interface{}) error, p *entity.Part) error {
	return sc(&p.ID, &p.SKU, &p.Name, &p.Unit, &p.StockQuantity, &p.ReorderPoint, &p.UnitCostCents, &p.CreatedAt, &p.UpdatedAt)
}

// GetPartByIDForUpdate 事务内加行锁查配件。
func (r *PartRepository) GetPartByIDForUpdate(ctx context.Context, id int64) (entity.Part, error) {
	var p entity.Part
	err := scanPart(r.db.QueryRowContext(ctx, `SELECT id,sku,name,unit,stock_quantity,reorder_point,unit_cost_cents,created_at,updated_at
		FROM parts WHERE id=? FOR UPDATE`, id).Scan, &p)
	if err != nil {
		return entity.Part{}, TranslateError(err)
	}
	return p, nil
}

// ListParts 分页过滤查询配件。
func (r *PartRepository) ListParts(ctx context.Context, page entity.Page, filter entity.Filter) ([]entity.Part, int64, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	if filter.Keyword != "" {
		where += " AND (sku LIKE ? OR name LIKE ?)"
		kw := "%" + filter.Keyword + "%"
		args = append(args, kw, kw)
	}
	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM parts "+where, args...).Scan(&total); err != nil {
		return nil, 0, TranslateError(err)
	}
	args = append(args, page.Limit, page.Offset)
	rows, err := r.db.QueryContext(ctx, `SELECT id,sku,name,unit,stock_quantity,reorder_point,unit_cost_cents,created_at,updated_at
		FROM parts `+where+` ORDER BY id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, 0, TranslateError(err)
	}
	defer rows.Close()
	var out []entity.Part
	for rows.Next() {
		var p entity.Part
		if err := scanPart(rows.Scan, &p); err != nil {
			return nil, 0, err
		}
		out = append(out, p)
	}
	return out, total, nil
}

// UpdateStock 更新库存为指定余额（配合行锁保证并发安全）。
func (r *PartRepository) UpdateStock(ctx context.Context, id int64, delta int64, balance int64) error {
	_, err := r.db.ExecContext(ctx, "UPDATE parts SET stock_quantity=?, updated_at=? WHERE id=?", balance, time.Now(), id)
	return TranslateError(err)
}

// AppendStockMovement 追加库存变动流水。
func (r *PartRepository) AppendStockMovement(ctx context.Context, m entity.PartStockMovement) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO part_stock_movements(part_id,change_quantity,reason,ref_order_id,balance_after,created_by)
		VALUES(?,?,?,?,?,?)`, m.PartID, m.ChangeQuantity, m.Reason, i64PtrPtr(m.RefOrderID), m.BalanceAfter, m.CreatedBy)
	return TranslateError(err)
}

// ListStockMovements 分页查库存流水。
func (r *PartRepository) ListStockMovements(ctx context.Context, partID int64, page entity.Page) ([]entity.PartStockMovement, int64, error) {
	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM part_stock_movements WHERE part_id=?", partID).Scan(&total); err != nil {
		return nil, 0, TranslateError(err)
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id,part_id,change_quantity,reason,ref_order_id,balance_after,created_by,created_at
		FROM part_stock_movements WHERE part_id=? ORDER BY id DESC LIMIT ? OFFSET ?`, partID, page.Limit, page.Offset)
	if err != nil {
		return nil, 0, TranslateError(err)
	}
	defer rows.Close()
	var out []entity.PartStockMovement
	for rows.Next() {
		var m entity.PartStockMovement
		var ref sql.NullInt64
		if err := rows.Scan(&m.ID, &m.PartID, &m.ChangeQuantity, &m.Reason, &ref, &m.BalanceAfter, &m.CreatedBy, &m.CreatedAt); err != nil {
			return nil, 0, err
		}
		if ref.Valid {
			v := ref.Int64
			m.RefOrderID = &v
		}
		out = append(out, m)
	}
	return out, total, nil
}

// ListLowStock 列出库存低于补货点的配件。
func (r *PartRepository) ListLowStock(ctx context.Context) ([]entity.Part, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,sku,name,unit,stock_quantity,reorder_point,unit_cost_cents,created_at,updated_at
		FROM parts WHERE stock_quantity<reorder_point ORDER BY stock_quantity`)
	if err != nil {
		return nil, TranslateError(err)
	}
	defer rows.Close()
	var out []entity.Part
	for rows.Next() {
		var p entity.Part
		if err := scanPart(rows.Scan, &p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}
