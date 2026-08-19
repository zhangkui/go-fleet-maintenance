package mysqlrepo

import (
	"database/sql"
	"time"
)

// nullTime 读取可空时间为指针，空值返回 nil。
func nullTime(s sql.NullTime) *time.Time {
	if s.Valid {
		t := s.Time
		return &t
	}
	return nil
}

// timePtrPtr 把 *time.Time 转为写库参数，nil 或零值返回 nil。
// 用于允许 NULL 的 DATE/TIMESTAMP 列。
func timePtrPtr(p *time.Time) interface{} {
	if p == nil {
		return nil
	}
	if p.IsZero() {
		return nil
	}
	return *p
}

// dateOnlyPtr 类似 timePtrPtr，但截断到日期精度。
func dateOnlyPtr(p *time.Time) interface{} {
	if p == nil {
		return nil
	}
	if p.IsZero() {
		return nil
	}
	return p.Format("2006-01-02")
}

// i64PtrPtr 把 *int64 转为写库参数，nil 或零值返回 nil。
func i64PtrPtr(p *int64) interface{} {
	if p == nil {
		return nil
	}
	if *p == 0 {
		return nil
	}
	return *p
}
