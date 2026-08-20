# BUG-029 Reproduction

## 现象
管理员禁用用户时接口返回了 500，预期是 204；随后再启用虽然返回 204，但两次操作对应的 user.toggle 审计数量仍然是 0。状态没有更新成功时不能记录成功审计，更新成功后也不能漏记，麻烦帮我把这两个问题一起处理。

## 触发命令

    go test ./internal/service -count=1 -run '^TestBug029_UserStatusAudit$'

## 缺陷基线结果
固定回归命令在缺陷分支退出码为 1，目标业务断言失败。

## 失效链路
涉及生产文件：internal/service/user.go、internal/repository/mysql/user.go。请求输入或任务状态在这些职责层之间传递时被错误转换、遗漏或使用，最终形成题面中的可见结果。

## Gold 机制变化

    +	s.audit.Record(ctx, actor, entity.AuditUserToggle, "user", id, map[string]string{"to": status})
    -	_, err := r.db.ExecContext(ctx, "UPDATE users SET status=? WHERE id=?", id, status)
    +	_, err := r.db.ExecContext(ctx, "UPDATE users SET status=? WHERE id=?", status, id)

## 正确结果
题面中的目标场景恢复正常，边界与对照场景保持原有行为；既有测试文件和断言不作修改。
