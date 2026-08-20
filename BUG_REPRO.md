# BUG-013 Reproduction

## 现象
运输任务完成时，我在请求里传了 `end_odometer_km=3000`，接口返回的结束里程却还是 0，重新查询数据库时这个字段也是 null。完成时间和任务状态都已经更新了，只有结束里程没有保存下来，麻烦帮我看看这个参数为什么丢了。

## 触发命令

    go test ./internal/service -count=1 -run '^TestBug013_TripCompletionOdometer$'

## 缺陷基线结果
固定回归命令在缺陷分支退出码为 1，目标业务断言失败。

## 失效链路
涉及生产文件：internal/service/trip.go、internal/repository/mysql/trip.go。请求输入或任务状态在这些职责层之间传递时被错误转换、遗漏或使用，最终形成题面中的可见结果。

## Gold 机制变化

    -		edo := t.StartOdometerKM
    +		edo := req.EndOdometerKM
    -	_, err := r.db.ExecContext(ctx, `UPDATE trips SET status=?, completed_at=?, completed_by=?, updated_at=? WHERE id=?`,
    -		entity.TripStatusCompleted, at, completedBy, at, id)
    +	_, err := r.db.ExecContext(ctx, `UPDATE trips SET status=?, end_odometer_km=?, completed_at=?, completed_by=?, updated_at=? WHERE id=?`,
    +		entity.TripStatusCompleted, endOdometer, at, completedBy, at, id)

## 正确结果
题面中的目标场景恢复正常，边界与对照场景保持原有行为；既有测试文件和断言不作修改。
