# BUG-021 Reproduction

## 现象
调整配件库存时，两个边界结果正好反了：库存减到 0 返回 400，继续减成负数反而返回 200。库存可以刚好用完，但不能小于 0，并发扣减也不能穿透这个下限，麻烦帮我把库存边界修正确。

## 触发命令

    go test ./internal/service -count=1 -run '^TestBug021_PartStockFloor$'

## 缺陷基线结果
固定回归命令在缺陷分支退出码为 1，目标业务断言失败。

## 失效链路
涉及生产文件：internal/service/part.go、internal/repository/mysql/part.go。请求输入或任务状态在这些职责层之间传递时被错误转换、遗漏或使用，最终形成题面中的可见结果。

## Gold 机制变化

    -		if newBalance <= 0 {
    +		if newBalance < 0 {
    -	_, err := r.db.ExecContext(ctx, "UPDATE parts SET stock_quantity=stock_quantity+? WHERE id=?", delta, id)
    +	_, err := r.db.ExecContext(ctx, "UPDATE parts SET stock_quantity=?, updated_at=? WHERE id=?", balance, time.Now(), id)

## 正确结果
题面中的目标场景恢复正常，边界与对照场景保持原有行为；既有测试文件和断言不作修改。
