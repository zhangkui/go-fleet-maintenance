package mysqlrepo

import (
	"context"
	"time"

	"github.com/zhangkui/go-fleet-maintenance/internal/domain/entity"
)

// AuditRepository 审计日志 MySQL 实现。
type AuditRepository struct {
	db DBTX
}

// NewAuditRepository 构造审计仓储。
func NewAuditRepository(db DBTX) *AuditRepository { return &AuditRepository{db: db} }

// Append 追加一条审计日志。
func (r *AuditRepository) Append(ctx context.Context, a entity.AuditLog) (entity.AuditLog, error) {
	res, err := r.db.ExecContext(ctx, `INSERT INTO audit_logs(actor_user_id,actor_name,action,resource_type,resource_id,detail,ip)
		VALUES(?,?,?,?,?,?,?)`, a.ActorUserID, a.ActorName, a.Action, a.ResourceType, a.ResourceID, a.Detail, a.IP)
	if err != nil {
		return entity.AuditLog{}, TranslateError(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return entity.AuditLog{}, err
	}
	a.ID = id
	a.CreatedAt = time.Now()
	return a, nil
}

// List 分页查询审计日志。
func (r *AuditRepository) List(ctx context.Context, page entity.Page, filter entity.Filter) ([]entity.AuditLog, int64, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	if filter.Status != "" {
		where += " AND resource_type=?"
		args = append(args, filter.Status)
	}
	if filter.Keyword != "" {
		where += " AND (actor_name LIKE ? OR action LIKE ? OR resource_type LIKE ?)"
		kw := "%" + filter.Keyword + "%"
		args = append(args, kw, kw, kw)
	}
	if !filter.From.IsZero() {
		where += " AND created_at>=?"
		args = append(args, filter.From)
	}
	if !filter.To.IsZero() {
		where += " AND created_at<=?"
		args = append(args, filter.To)
	}
	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM audit_logs "+where, args...).Scan(&total); err != nil {
		return nil, 0, TranslateError(err)
	}
	args = append(args, page.Limit, page.Offset)
	rows, err := r.db.QueryContext(ctx, `SELECT id,actor_user_id,actor_name,action,resource_type,resource_id,detail,ip,created_at
		FROM audit_logs `+where+` ORDER BY id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, 0, TranslateError(err)
	}
	defer rows.Close()
	var out []entity.AuditLog
	for rows.Next() {
		var a entity.AuditLog
		if err := rows.Scan(&a.ID, &a.ActorUserID, &a.ActorName, &a.Action, &a.ResourceType, &a.ResourceID, &a.Detail, &a.IP, &a.CreatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, a)
	}
	return out, total, nil
}
