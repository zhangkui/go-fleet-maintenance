# BUG-022 Reproduction

## 现象
低库存查询没有返回库存量刚好等于 reorder_point 的配件。低于补货点的记录能正常查到，但等值场景期望 1 条、实际是 0 条；补货边界应该包含等于的情况，麻烦帮我修一下。

## 触发命令

    go test ./internal/service -count=1 -run '^TestBug022_PartReorderBoundary$'

## 缺陷基线结果
固定回归命令在缺陷分支退出码为 1，目标业务断言失败。

## 失效链路
涉及生产文件：internal/service/part.go、internal/repository/mysql/part.go。请求输入或任务状态在这些职责层之间传递时被错误转换、遗漏或使用，最终形成题面中的可见结果。

## Gold 机制变化

    -	parts, err := s.repo.ListLowStock(ctx)
    -	if err != nil {
    -		return nil, err
    -	}
    -	filtered := make([]entity.Part, 0, len(parts))
    -	for _, part := range parts {
    -		if part.StockQuantity < part.ReorderPoint {
    -			filtered = append(filtered, part)

## 正确结果
题面中的目标场景恢复正常，边界与对照场景保持原有行为；既有测试文件和断言不作修改。
