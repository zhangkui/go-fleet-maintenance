package mysqlrepo

import (
	"context"
	"database/sql"
	"time"

	"github.com/zhangkui/go-fleet-maintenance/internal/domain/entity"
)

// ReminderRepository 到期提醒 MySQL 实现。
type ReminderRepository struct {
	db DBTX
}

// NewReminderRepository 构造提醒仓储。
func NewReminderRepository(db DBTX) *ReminderRepository { return &ReminderRepository{db: db} }

// CreateReminder 创建提醒。
func (r *ReminderRepository) CreateReminder(ctx context.Context, rem entity.Reminder) (entity.Reminder, error) {
	res, err := r.db.ExecContext(ctx, `INSERT INTO reminders(entity_type,entity_id,due_at,message,status)
		VALUES(?,?,?,?,'pending')`, rem.EntityType, rem.EntityID, rem.DueAt, rem.Message)
	if err != nil {
		return entity.Reminder{}, TranslateError(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return entity.Reminder{}, err
	}
	rem.ID = id
	rem.Status = entity.ReminderStatusPending
	rem.CreatedAt = time.Now()
	return rem, nil
}

// ListPendingReminders 列出指定时间前未处理的提醒。
func (r *ReminderRepository) ListPendingReminders(ctx context.Context, before time.Time) ([]entity.Reminder, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,entity_type,entity_id,due_at,message,status,created_at,sent_at
		FROM reminders WHERE status='pending' AND due_at<=? ORDER BY due_at`, before)
	if err != nil {
		return nil, TranslateError(err)
	}
	defer rows.Close()
	var out []entity.Reminder
	for rows.Next() {
		var rem entity.Reminder
		var sent sql.NullTime
		if err := rows.Scan(&rem.ID, &rem.EntityType, &rem.EntityID, &rem.DueAt, &rem.Message, &rem.Status, &rem.CreatedAt, &sent); err != nil {
			return nil, err
		}
		rem.SentAt = nullTime(sent)
		out = append(out, rem)
	}
	return out, nil
}

// MarkSent 标记提醒已发送。
func (r *ReminderRepository) MarkSent(ctx context.Context, id int64, at time.Time) error {
	_, err := r.db.ExecContext(ctx, "UPDATE reminders SET status='sent', sent_at=? WHERE id=?", at, id)
	return TranslateError(err)
}

// ListReminders 分页查询提醒。
func (r *ReminderRepository) ListReminders(ctx context.Context, page entity.Page, filter entity.Filter) ([]entity.Reminder, int64, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	if filter.Status != "" {
		where += " AND status=?"
		args = append(args, filter.Status)
	}
	if filter.Kind != "" {
		where += " AND entity_type=?"
		args = append(args, filter.Kind)
	}
	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM reminders "+where, args...).Scan(&total); err != nil {
		return nil, 0, TranslateError(err)
	}
	args = append(args, page.Limit, page.Offset)
	rows, err := r.db.QueryContext(ctx, `SELECT id,entity_type,entity_id,due_at,message,status,created_at,sent_at
		FROM reminders `+where+` ORDER BY id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, 0, TranslateError(err)
	}
	defer rows.Close()
	var out []entity.Reminder
	for rows.Next() {
		var rem entity.Reminder
		var sent sql.NullTime
		if err := rows.Scan(&rem.ID, &rem.EntityType, &rem.EntityID, &rem.DueAt, &rem.Message, &rem.Status, &rem.CreatedAt, &sent); err != nil {
			return nil, 0, err
		}
		rem.SentAt = nullTime(sent)
		out = append(out, rem)
	}
	return out, total, nil
}
