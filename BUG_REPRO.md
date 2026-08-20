# BUG-020 Reproduction

## 现象
同一辆车的同一条维保策略连续扫描三次，第二次和第三次仍然各新增了一张工单，测试里两次都是期望新增 0、实际新增 1。已有进行中工单时也不应该再生成。先别修，查完把结论告诉我；请说明重复扫描为何没识别到已有工单，以及并发触发时还存在哪些重复风险。

## 触发命令

    go test ./internal/service -count=1 -run '^TestBug020_MaintenancePolicyDedup$'

## 缺陷基线结果
固定回归命令在缺陷分支退出码为 1，目标业务断言失败。

## 失效链路
涉及生产文件：internal/service/maintenance.go、internal/repository/mysql/maintenance.go。请求输入或任务状态在这些职责层之间传递时被错误转换、遗漏或使用，最终形成题面中的可见结果。

## Gold 机制变化

    +		didCreate := false
    -			if !open {
    +			if open {
    -			if err != nil && !isConflict(err) {
    +			if err != nil {
    +				if isConflict(err) {
    +					return nil
    +				}

## 正确结果
题面中的目标场景恢复正常，边界与对照场景保持原有行为；既有测试文件和断言不作修改。
