package entity

import "time"

// Reminder 到期提醒实体。
type Reminder struct {
	ID         int64      `json:"id"`
	EntityType string     `json:"entity_type"`
	EntityID   int64      `json:"entity_id"`
	DueAt      time.Time  `json:"due_at"`
	Message    string     `json:"message"`
	Status     string     `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
	SentAt     *time.Time `json:"sent_at,omitempty"`
}

// Notification 站内通知实体。
type Notification struct {
	ID        int64      `json:"id"`
	UserID    int64      `json:"user_id"`
	Kind      string     `json:"kind"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	ReadAt    *time.Time `json:"read_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// ReminderScanResult 一次到期扫描的汇总结果。
type ReminderScanResult struct {
	Insurance   int `json:"insurance"`
	Inspection  int `json:"inspection"`
	License     int `json:"license"`
	Maintenance int `json:"maintenance"`
	Created     int `json:"created"`
}
