package mysqlrepo

import (
	"context"
	"database/sql"
	"time"

	"github.com/zhangkui/go-fleet-maintenance/internal/domain/entity"
)

// UserRepository 用户 MySQL 实现。
type UserRepository struct {
	db DBTX
}

// NewUserRepository 构造用户仓储。
func NewUserRepository(db DBTX) *UserRepository { return &UserRepository{db: db} }

// CreateUser 创建用户。
func (r *UserRepository) CreateUser(ctx context.Context, u entity.User) (entity.User, error) {
	res, err := r.db.ExecContext(ctx, `INSERT INTO users(username,password_hash,email,full_name,status)
		VALUES(?,?,?,?,?)`, u.Username, u.PasswordHash, u.Email, u.FullName, entity.UserStatusActive)
	if err != nil {
		return entity.User{}, TranslateError(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return entity.User{}, err
	}
	u.ID = id
	u.Status = entity.UserStatusActive
	u.CreatedAt = time.Now()
	u.UpdatedAt = u.CreatedAt
	return u, nil
}

// GetUserByID 按 ID 查用户。
func (r *UserRepository) GetUserByID(ctx context.Context, id int64) (entity.User, error) {
	var u entity.User
	var lastLogin, locked sql.NullTime
	err := r.db.QueryRowContext(ctx, `SELECT id,username,password_hash,email,full_name,status,
		failed_login_count,locked_until,last_login_at,created_at,updated_at FROM users WHERE id=?`, id).Scan(
		&u.ID, &u.Username, &u.PasswordHash, &u.Email, &u.FullName, &u.Status,
		&u.FailedLoginCount, &locked, &lastLogin, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return entity.User{}, TranslateError(err)
	}
	u.LockedUntil = nullTime(locked)
	u.LastLoginAt = nullTime(lastLogin)
	return u, nil
}

// GetUserByUsername 按用户名查用户。
func (r *UserRepository) GetUserByUsername(ctx context.Context, username string) (entity.User, error) {
	var u entity.User
	var lastLogin, locked sql.NullTime
	err := r.db.QueryRowContext(ctx, `SELECT id,username,password_hash,email,full_name,status,
		failed_login_count,locked_until,last_login_at,created_at,updated_at FROM users WHERE username=?`, username).Scan(
		&u.ID, &u.Username, &u.PasswordHash, &u.Email, &u.FullName, &u.Status,
		&u.FailedLoginCount, &locked, &lastLogin, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return entity.User{}, TranslateError(err)
	}
	u.LockedUntil = nullTime(locked)
	u.LastLoginAt = nullTime(lastLogin)
	return u, nil
}

// ListUsers 分页列出用户。
func (r *UserRepository) ListUsers(ctx context.Context, page entity.Page, filter entity.Filter) ([]entity.User, int64, error) {
	where := "WHERE status='active'"
	args := []interface{}{}
	if filter.Status != "" {
		where = "WHERE status=?"
		args = append(args, filter.Status)
	}
	if filter.Keyword != "" {
		where += " AND (username LIKE ? OR full_name LIKE ? OR email LIKE ?)"
		kw := "%" + filter.Keyword + "%"
		args = append(args, kw, kw, kw)
	}
	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users "+where, args...).Scan(&total); err != nil {
		return nil, 0, TranslateError(err)
	}
	args = append(args, page.Limit, page.Offset)
	rows, err := r.db.QueryContext(ctx, `SELECT id,username,email,full_name,status,last_login_at,created_at,updated_at
		FROM users `+where+` ORDER BY id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, 0, TranslateError(err)
	}
	defer rows.Close()
	var out []entity.User
	for rows.Next() {
		var u entity.User
		var lastLogin sql.NullTime
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.FullName, &u.Status, &lastLogin, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, 0, err
		}
		u.LastLoginAt = nullTime(lastLogin)
		out = append(out, u)
	}
	return out, total, nil
}

// UpdateUserStatus 更新启停状态。
func (r *UserRepository) UpdateUserStatus(ctx context.Context, id int64, status string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE users SET status=? WHERE id=?", id, status)
	return TranslateError(err)
}

// UpdateUserPassword 更新密码哈希。
func (r *UserRepository) UpdateUserPassword(ctx context.Context, id int64, hash string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE users SET password_hash=? WHERE id=?", hash, id)
	return TranslateError(err)
}

// UpdateLastLogin 更新最后登录时间并清零失败计数。
func (r *UserRepository) UpdateLastLogin(ctx context.Context, id int64, at time.Time) error {
	_, err := r.db.ExecContext(ctx, "UPDATE users SET last_login_at=?, failed_login_count=0, locked_until=NULL WHERE id=?", at, id)
	return TranslateError(err)
}

// IncFailedLogin 自增失败登录计数并返回当前计数。
func (r *UserRepository) IncFailedLogin(ctx context.Context, id int64) (int, error) {
	var old int
	if err := r.db.QueryRowContext(ctx, "SELECT failed_login_count FROM users WHERE id=?", id).Scan(&old); err != nil {
		return 0, TranslateError(err)
	}
	if _, err := r.db.ExecContext(ctx, "UPDATE users SET failed_login_count=failed_login_count+1 WHERE id=?", id); err != nil {
		return 0, TranslateError(err)
	}
	return old, nil
}

// ResetFailedLogin 清零失败登录计数并解锁。
func (r *UserRepository) ResetFailedLogin(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "UPDATE users SET failed_login_count=0, locked_until=NULL WHERE id=?", id)
	return TranslateError(err)
}

// LockUser 锁定用户到指定时间。
func (r *UserRepository) LockUser(ctx context.Context, id int64, until time.Time) error {
	_, err := r.db.ExecContext(ctx, "UPDATE users SET locked_until=? WHERE id=?", until, id)
	return TranslateError(err)
}
