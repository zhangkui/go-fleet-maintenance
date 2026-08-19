// Package mysqlrepo 是 repository 接口的 MySQL 实现，并包含事务边界。
package mysqlrepo

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/zhangkui/go-fleet-maintenance/internal/domain"
	"github.com/zhangkui/go-fleet-maintenance/internal/platform/mysql"
	"github.com/zhangkui/go-fleet-maintenance/internal/repository"
)

// DBTX 别名，等价于 mysql.DBTX，被 *sql.DB 与 *sql.Tx 同时满足。
type DBTX = mysql.DBTX

// Transactor 事务边界实现。
type Transactor struct {
	db *sql.DB
}

// NewTransactor 创建事务管理器。
func NewTransactor(db *sql.DB) *Transactor { return &Transactor{db: db} }

// DB 返回底层连接池，用于构建非事务仓储集合。
func (t *Transactor) DB() *sql.DB { return t.db }

// BuildStores 构造绑定到给定 DBTX 的全量仓储集合。
func BuildStores(db DBTX) repository.Stores {
	return repository.Stores{
		Users:       NewUserRepository(db),
		Roles:       NewRoleRepository(db),
		Permissions: NewPermissionRepository(db),
		Sessions:    NewSessionRepository(db),
		Audit:       NewAuditRepository(db),
		Vehicles:    NewVehicleRepository(db),
		Drivers:     NewDriverRepository(db),
		Trips:       NewTripRepository(db),
		Fuel:        NewFuelRepository(db),
		Maintenance: NewMaintenanceRepository(db),
		Parts:       NewPartRepository(db),
		Reminders:   NewReminderRepository(db),
		Reports:     NewReportRepository(db),
	}
}

// Stores 返回绑定到连接池的仓储集合（用于只读与简单写）。
func (t *Transactor) Stores() repository.Stores { return BuildStores(t.db) }

// WithinTx 在事务中执行 fn，传入绑定到事务的仓储集合。
// fn 内部不得再次调用本方法（无嵌套复用）；service 层每个事务方法自包含。
func (t *Transactor) WithinTx(ctx context.Context, fn func(repository.Stores) error) error {
	tx, err := t.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.NewCoded("internal_error", "开启事务失败: "+err.Error(), err)
	}
	if err := fn(BuildStores(tx)); err != nil {
		if rerr := tx.Rollback(); rerr != nil && !errors.Is(rerr, sql.ErrTxDone) {
			return errors.Join(err, rerr)
		}
		return err
	}
	if err := tx.Commit(); err != nil {
		return domain.NewCoded("internal_error", "提交事务失败: "+err.Error(), err)
	}
	return nil
}

// TranslateError 把 MySQL 驱动错误翻译为领域错误。
func TranslateError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrNotFound
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "Duplicate entry"):
		return domain.ErrConflict
	case strings.Contains(msg, "Cannot add or update a child row"):
		return domain.NewCoded("validation_error", "关联资源不存在", domain.ErrValidation)
	}
	return err
}
