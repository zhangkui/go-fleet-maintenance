// Package migration 提供版本化、幂等的数据库迁移执行器。
package migration

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"strings"
)

//go:embed sql/*.sql
var fs embed.FS

// Runner 迁移执行器。
type Runner struct {
	db *sql.DB
}

// New 创建迁移执行器。
func New(db *sql.DB) *Runner { return &Runner{db: db} }

// EnsureSchemaMigrations 建版本表，幂等。
func (r *Runner) EnsureSchemaMigrations(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version VARCHAR(64) PRIMARY KEY,
		applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)
	return err
}

// Run 执行全部内嵌迁移，已应用过的跳过。
func (r *Runner) Run(ctx context.Context) ([]string, error) {
	if err := r.EnsureSchemaMigrations(ctx); err != nil {
		return nil, fmt.Errorf("建版本表失败: %w", err)
	}
	names, err := fs.ReadDir("sql")
	if err != nil {
		return nil, fmt.Errorf("读取迁移目录失败: %w", err)
	}
	applied := make(map[string]bool)
	rows, err := r.db.QueryContext(ctx, "SELECT version FROM schema_migrations")
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			rows.Close()
			return nil, err
		}
		applied[v] = true
	}
	rows.Close()

	var ordered []string
	for _, f := range names {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".sql") {
			ordered = append(ordered, f.Name())
		}
	}
	sort.Strings(ordered)

	var ran []string
	for _, name := range ordered {
		version := strings.TrimSuffix(name, ".sql")
		if applied[version] {
			continue
		}
		stmts, err := fs.ReadFile("sql/" + name)
		if err != nil {
			return ran, fmt.Errorf("读取 %s 失败: %w", name, err)
		}
		tx, err := r.db.BeginTx(ctx, nil)
		if err != nil {
			return ran, err
		}
		// 按分号拆分为单条语句逐条执行，避免依赖 multiStatements 驱动参数。
		for _, stmt := range splitStatements(string(stmts)) {
			if strings.TrimSpace(stmt) == "" {
				continue
			}
			if _, err := tx.ExecContext(ctx, stmt); err != nil {
				tx.Rollback()
				return ran, fmt.Errorf("执行 %s 失败: %w", name, err)
			}
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO schema_migrations(version) VALUES(?)", version); err != nil {
			tx.Rollback()
			return ran, fmt.Errorf("记录版本 %s 失败: %w", version, err)
		}
		if err := tx.Commit(); err != nil {
			return ran, err
		}
		ran = append(ran, version)
	}
	return ran, nil
}

// splitStatements 按分号拆分 SQL 文件为单条语句。
// 当前迁移文件不含触发器/存储过程/分号字面量，按分号拆分安全。
func splitStatements(src string) []string {
	// 先按行去除 -- 注释行。
	var cleaned []string
	for _, line := range strings.Split(src, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "--") {
			continue
		}
		cleaned = append(cleaned, line)
	}
	joined := strings.Join(cleaned, "\n")
	parts := strings.Split(joined, ";")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
