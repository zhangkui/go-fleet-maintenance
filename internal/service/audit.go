// Package service 实现全部业务编排、事务与不变量校验。
package service

import (
	"context"
	"encoding/json"

	"github.com/zhangkui/go-fleet-maintenance/internal/domain/entity"
	"github.com/zhangkui/go-fleet-maintenance/internal/repository"
)

// AuditService 审计写入服务。
type AuditService struct {
	repo repository.AuditRepository
}

// NewAuditService 构造审计服务。
func NewAuditService(repo repository.AuditRepository) *AuditService {
	return &AuditService{repo: repo}
}

// Record 写入审计日志，detail 自动序列化。
func (s *AuditService) Record(ctx context.Context, actor entity.AuditActor, action, resourceType string, resourceID int64, detail interface{}) {
	var detailStr string
	if detail != nil {
		if b, err := json.Marshal(detail); err == nil {
			detailStr = string(b)
		}
	}
	// 审计日志失败不应中断主流程，但需记录。
	_, _ = s.repo.Append(ctx, entity.AuditLog{
		ActorUserID:  actor.UserID,
		ActorName:    actor.Username,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Detail:       detailStr,
		IP:           actor.IP,
	})
}
