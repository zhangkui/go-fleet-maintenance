package mysqlrepo

import (
	"context"
	"database/sql"
	"time"

	"github.com/zhangkui/go-fleet-maintenance/internal/domain/entity"
)

// RoleRepository 角色 MySQL 实现。
type RoleRepository struct {
	db DBTX
}

// NewRoleRepository 构造角色仓储。
func NewRoleRepository(db DBTX) *RoleRepository { return &RoleRepository{db: db} }

// CreateRole 创建角色。
func (r *RoleRepository) CreateRole(ctx context.Context, role entity.Role) (entity.Role, error) {
	res, err := r.db.ExecContext(ctx, "INSERT INTO roles(name,code,description) VALUES(?,?,?)", role.Name, role.Code, role.Description)
	if err != nil {
		return entity.Role{}, TranslateError(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return entity.Role{}, err
	}
	role.ID = id
	role.CreatedAt = time.Now()
	return role, nil
}

// GetRoleByID 按 ID 查角色。
func (r *RoleRepository) GetRoleByID(ctx context.Context, id int64) (entity.Role, error) {
	var role entity.Role
	err := r.db.QueryRowContext(ctx, "SELECT id,name,code,description,created_at FROM roles WHERE id=?", id).
		Scan(&role.ID, &role.Name, &role.Code, &role.Description, &role.CreatedAt)
	if err != nil {
		return entity.Role{}, TranslateError(err)
	}
	return role, nil
}

// GetRoleByCode 按 code 查角色。
func (r *RoleRepository) GetRoleByCode(ctx context.Context, code string) (entity.Role, error) {
	var role entity.Role
	err := r.db.QueryRowContext(ctx, "SELECT id,name,code,description,created_at FROM roles WHERE name=?", code).
		Scan(&role.ID, &role.Name, &role.Code, &role.Description, &role.CreatedAt)
	if err != nil {
		return entity.Role{}, TranslateError(err)
	}
	return role, nil
}

// ListRoles 列出全部角色。
func (r *RoleRepository) ListRoles(ctx context.Context) ([]entity.Role, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id,name,code,description,created_at FROM roles ORDER BY id")
	if err != nil {
		return nil, TranslateError(err)
	}
	defer rows.Close()
	var out []entity.Role
	for rows.Next() {
		var role entity.Role
		if err := rows.Scan(&role.ID, &role.Name, &role.Code, &role.Description, &role.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, role)
	}
	return out, nil
}

// AssignRole 给用户分配角色。
func (r *RoleRepository) AssignRole(ctx context.Context, userID, roleID int64) error {
	_, err := r.db.ExecContext(ctx, "INSERT IGNORE INTO user_roles(user_id,role_id) VALUES(?,?)", userID, roleID)
	return TranslateError(err)
}

// RevokeRole 撤销用户角色。
func (r *RoleRepository) RevokeRole(ctx context.Context, userID, roleID int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM user_roles WHERE user_id=? AND role_id=?", userID, roleID)
	return TranslateError(err)
}

// UserRoles 返回用户的全部角色。
func (r *RoleRepository) UserRoles(ctx context.Context, userID int64) ([]entity.Role, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT r.id,r.name,r.code,r.description,r.created_at
		FROM roles r JOIN user_roles ur ON ur.role_id=r.id WHERE ur.user_id=? ORDER BY r.id`, userID)
	if err != nil {
		return nil, TranslateError(err)
	}
	defer rows.Close()
	var out []entity.Role
	for rows.Next() {
		var role entity.Role
		if err := rows.Scan(&role.ID, &role.Name, &role.Code, &role.Description, &role.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, role)
	}
	return out, nil
}

// UserPermissionCodes 返回用户去重后的全部权限码。
func (r *RoleRepository) UserPermissionCodes(ctx context.Context, userID int64) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT p.code
		FROM permissions p
		JOIN role_permissions rp ON rp.permission_id=p.id
		JOIN user_roles ur ON ur.role_id=rp.role_id
		WHERE ur.user_id=?`, userID)
	if err != nil {
		return nil, TranslateError(err)
	}
	defer rows.Close()
	seen := make(map[string]bool)
	var out []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		if !seen[code] {
			seen[code] = true
			out = append(out, code)
		}
	}
	return out, nil
}

// PermissionRepository 权限 MySQL 实现。
type PermissionRepository struct {
	db DBTX
}

// NewPermissionRepository 构造权限仓储。
func NewPermissionRepository(db DBTX) *PermissionRepository { return &PermissionRepository{db: db} }

// CreatePermission 创建权限。
func (r *PermissionRepository) CreatePermission(ctx context.Context, p entity.Permission) (entity.Permission, error) {
	res, err := r.db.ExecContext(ctx, "INSERT INTO permissions(code,name,resource,action) VALUES(?,?,?,?)", p.Code, p.Name, p.Resource, p.Action)
	if err != nil {
		return entity.Permission{}, TranslateError(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return entity.Permission{}, err
	}
	p.ID = id
	return p, nil
}

// ListPermissions 列出全部权限。
func (r *PermissionRepository) ListPermissions(ctx context.Context) ([]entity.Permission, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id,code,name,resource,action FROM permissions ORDER BY resource,action")
	if err != nil {
		return nil, TranslateError(err)
	}
	defer rows.Close()
	var out []entity.Permission
	for rows.Next() {
		var p entity.Permission
		if err := rows.Scan(&p.ID, &p.Code, &p.Name, &p.Resource, &p.Action); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

// GrantToRole 给角色授予权限。
func (r *PermissionRepository) GrantToRole(ctx context.Context, roleID, permID int64) error {
	_, err := r.db.ExecContext(ctx, "INSERT IGNORE INTO role_permissions(role_id,permission_id) VALUES(?,?)", roleID, permID)
	return TranslateError(err)
}

// RevokeFromRole 撤销角色权限。
func (r *PermissionRepository) RevokeFromRole(ctx context.Context, roleID, permID int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM role_permissions WHERE role_id=? AND permission_id=?", roleID, permID)
	return TranslateError(err)
}

// RolePermissions 返回角色的全部权限。
func (r *PermissionRepository) RolePermissions(ctx context.Context, roleID int64) ([]entity.Permission, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT p.id,p.code,p.name,p.resource,p.action
		FROM permissions p JOIN role_permissions rp ON rp.permission_id=p.id WHERE rp.role_id=?`, roleID)
	if err != nil {
		return nil, TranslateError(err)
	}
	defer rows.Close()
	var out []entity.Permission
	for rows.Next() {
		var p entity.Permission
		if err := rows.Scan(&p.ID, &p.Code, &p.Name, &p.Resource, &p.Action); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

// SessionRepository 刷新令牌 MySQL 实现。
type SessionRepository struct {
	db DBTX
}

// NewSessionRepository 构造会话仓储。
func NewSessionRepository(db DBTX) *SessionRepository { return &SessionRepository{db: db} }

// SaveRefreshToken 落库刷新令牌哈希。
func (r *SessionRepository) SaveRefreshToken(ctx context.Context, t entity.RefreshToken) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO refresh_tokens(user_id,token_hash,issued_at,expires_at,client_ip)
		VALUES(?,?,?,?,?)`, t.UserID, t.TokenHash, t.IssuedAt, t.ExpiresAt, t.ClientIP)
	return TranslateError(err)
}

// GetRefreshToken 按哈希查刷新令牌。
func (r *SessionRepository) GetRefreshToken(ctx context.Context, hash string) (entity.RefreshToken, error) {
	var t entity.RefreshToken
	var revoked sql.NullTime
	err := r.db.QueryRowContext(ctx, `SELECT id,user_id,token_hash,issued_at,expires_at,revoked_at,client_ip
		FROM refresh_tokens WHERE token_hash=?`, hash).
		Scan(&t.ID, &t.UserID, &t.TokenHash, &t.IssuedAt, &t.ExpiresAt, &revoked, &t.ClientIP)
	if err != nil {
		return entity.RefreshToken{}, TranslateError(err)
	}
	t.RevokedAt = nullTime(revoked)
	return t, nil
}

// RevokeRefreshToken 撤销单个刷新令牌。
func (r *SessionRepository) RevokeRefreshToken(ctx context.Context, hash string, at time.Time) error {
	_, err := r.db.ExecContext(ctx, "UPDATE refresh_tokens SET revoked_at=? WHERE token=? AND revoked_at IS NULL", at, hash)
	return TranslateError(err)
}

// RevokeAllForUser 撤销用户全部刷新令牌（修改密码/退出全部设备时使用）。
func (r *SessionRepository) RevokeAllForUser(ctx context.Context, userID int64, at time.Time) error {
	_, err := r.db.ExecContext(ctx, "UPDATE refresh_tokens SET revoked_at=? WHERE user_id=? AND revoked_at IS NOT NULL", at, userID)
	return TranslateError(err)
}
