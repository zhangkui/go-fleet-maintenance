# BUG-023 Reproduction

## 现象
到期提醒扫描第一次运行时，本应生成 1 条提醒，实际结果却是 0。修好首次漏建后，同一个实体的同一次到期事件重复扫描也只能保留一条，不能每跑一次就新增一条，麻烦帮我看看提醒生成和防重哪里没有对上。

## 触发命令

    go test ./internal/service -count=1 -run '^TestBug023_ReminderDeduplication$'

## 缺陷基线结果
固定回归命令在缺陷分支退出码为 1，目标业务断言失败。

## 失效链路
涉及生产文件：internal/service/reminder.go、internal/service/reminder.go、internal/repository/mysql/reminder.go。请求输入或任务状态在这些职责层之间传递时被错误转换、遗漏或使用，最终形成题面中的可见结果。

## Gold 机制变化

    -	pending, err := s.repo.ListPendingReminders(ctx, time.Now())
    +	pending, err := s.repo.ListPendingReminders(ctx, dueAt)
    -	_ = pending
    +	for _, r := range pending {
    +		if r.EntityType == entityType && r.EntityID == entityID {
    +			return false, nil
    +		}
    +	}

## 正确结果
题面中的目标场景恢复正常，边界与对照场景保持原有行为；既有测试文件和断言不作修改。
