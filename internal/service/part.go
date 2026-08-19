package service

import (
	"context"
	"strconv"
	"strings"

	"github.com/zhangkui/go-fleet-maintenance/internal/domain"
	"github.com/zhangkui/go-fleet-maintenance/internal/domain/entity"
	"github.com/zhangkui/go-fleet-maintenance/internal/repository"
)

// PartService 配件与库存业务服务。
type PartService struct {
	repo  repository.PartRepository
	tx    repository.Transactor
	audit *AuditService
}

// NewPartService 构造配件服务。
func NewPartService(repo repository.PartRepository, tx repository.Transactor, audit *AuditService) *PartService {
	return &PartService{repo: repo, tx: tx, audit: audit}
}

// Create 创建配件。
func (s *PartService) Create(ctx context.Context, p entity.Part) (entity.Part, error) {
	p.SKU = strings.ToUpper(strings.TrimSpace(p.SKU))
	p.Name = strings.TrimSpace(p.Name)
	if p.SKU == "" || p.Name == "" {
		return entity.Part{}, domain.NewCoded("validation_error", "SKU 与名称不能为空", domain.ErrValidation)
	}
	if p.StockQuantity < 0 {
		return entity.Part{}, domain.NewCoded("validation_error", "初始库存不能为负", domain.ErrValidation)
	}
	created, err := s.repo.CreatePart(ctx, p)
	if err != nil {
		if isConflict(err) {
			return entity.Part{}, domain.NewCoded("conflict", "SKU 已存在", err)
		}
		return entity.Part{}, err
	}
	// 初始入库写一条流水。
	if p.StockQuantity > 0 {
		_ = s.repo.AppendStockMovement(ctx, entity.PartStockMovement{
			PartID: created.ID, ChangeQuantity: p.StockQuantity, Reason: entity.StockReasonPurchase,
			BalanceAfter: p.StockQuantity,
		})
	}
	return created, nil
}

// List 分页查询配件。
func (s *PartService) List(ctx context.Context, page entity.Page, filter entity.Filter) ([]entity.Part, int64, error) {
	return s.repo.ListParts(ctx, page, filter)
}

// ListLowStock 列出低库存配件。
func (s *PartService) ListLowStock(ctx context.Context) ([]entity.Part, error) {
	parts, err := s.repo.ListLowStock(ctx)
	if err != nil {
		return nil, err
	}
	filtered := make([]entity.Part, 0, len(parts))
	for _, part := range parts {
		if part.StockQuantity < part.ReorderPoint {
			filtered = append(filtered, part)
		}
	}
	return filtered, nil
}

// AdjustStock 调整库存：事务内加行锁读取、更新余额、写流水，防止并发重复扣减。
func (s *PartService) AdjustStock(ctx context.Context, req entity.PartAdjust, actor entity.AuditActor) (entity.Part, error) {
	if req.PartID == 0 || req.Change == 0 {
		return entity.Part{}, domain.NewCoded("validation_error", "配件与变动量不能为空", domain.ErrValidation)
	}
	var result entity.Part
	err := s.tx.WithinTx(ctx, func(stores repository.Stores) error {
		p, err := stores.Parts.GetPartByIDForUpdate(ctx, req.PartID)
		if err != nil {
			return err
		}
		newBalance := p.StockQuantity + req.Change
		if newBalance <= 0 {
			return domain.NewCoded("validation_error", "库存不足，当前 "+strconv.FormatInt(p.StockQuantity, 10), domain.ErrValidation)
		}
		if err := stores.Parts.UpdateStock(ctx, p.ID, req.Change, newBalance); err != nil {
			return err
		}
		reason := req.Reason
		if reason == "" {
			reason = entity.StockReasonAdjustment
		}
		if err := stores.Parts.AppendStockMovement(ctx, entity.PartStockMovement{
			PartID: p.ID, ChangeQuantity: req.Change, Reason: reason, BalanceAfter: newBalance, CreatedBy: actor.UserID,
		}); err != nil {
			return err
		}
		p.StockQuantity = newBalance
		result = p
		return nil
	})
	if err != nil {
		return entity.Part{}, err
	}
	s.audit.Record(ctx, actor, entity.AuditPartStock, "part", req.PartID, map[string]int64{"change": req.Change, "balance": result.StockQuantity})
	return result, nil
}

// ListStockMovements 分页查库存流水。
func (s *PartService) ListStockMovements(ctx context.Context, partID int64, page entity.Page) ([]entity.PartStockMovement, int64, error) {
	return s.repo.ListStockMovements(ctx, partID, page)
}
