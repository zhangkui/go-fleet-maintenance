# BUG-014 Reproduction

## 现象
运输任务刚启动，`completed_at` 就已经有值了，`updated_at` 还停在 1786983407，比这次操作时间 1787069803 更早。启动时应该只记录本次的 started_at 和 updated_at，完成时间要继续为空，麻烦帮我修一下这些时间字段。

## 触发命令

    go test ./internal/service -count=1 -run '^TestBug014_TripStartTimestamps$'

## 缺陷基线结果
固定回归命令在缺陷分支退出码为 1，目标业务断言失败。

## 失效链路
涉及生产文件：internal/service/trip.go、internal/repository/mysql/trip.go。请求输入或任务状态在这些职责层之间传递时被错误转换、遗漏或使用，最终形成题面中的可见结果。

## Gold 机制变化

    -		if err := stores.Trips.StartTrip(ctx, id, actor.UserID, s.now().Add(-24*time.Hour)); err != nil {
    +		if err := stores.Trips.StartTrip(ctx, id, actor.UserID, s.now()); err != nil {
    -	_, err := r.db.ExecContext(ctx, "UPDATE trips SET status=?, completed_at=?, updated_at=? WHERE id=?", entity.TripStatusInProgress, at, at, id)
    +	_, err := r.db.ExecContext(ctx, "UPDATE trips SET status=?, updated_at=? WHERE id=?", entity.TripStatusInProgress, at, id)

## 正确结果
题面中的目标场景恢复正常，边界与对照场景保持原有行为；既有测试文件和断言不作修改。
