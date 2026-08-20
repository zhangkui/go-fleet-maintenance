# BUG-012 Reproduction

## 现象
违章接口接收的罚款单位是分，但我传 20000 后查出来只有 2，19999 变成 1，100 甚至变成了 0，只有传 0 时结果看起来正常。只分析原因并给我结论，项目先不要改；麻烦把金额在哪个环节被缩小、请求进入后每一步的实际数值和单位查清楚。

## 触发命令

    go test ./internal/service -count=1 -run '^TestBug012_DriverViolationFineCents$'

## 缺陷基线结果
固定回归命令在缺陷分支退出码为 1，目标业务断言失败。

## 失效链路
涉及生产文件：internal/transport/http/handler/driver.go、internal/service/driver.go。请求输入或任务状态在这些职责层之间传递时被错误转换、遗漏或使用，最终形成题面中的可见结果。

## Gold 机制变化

    -	v.FineCents = v.FineCents / 100
    -	// 罚款金额由元转分时除以100（正确应乘100）。
    -	v.FineCents = v.FineCents / 100

## 正确结果
题面中的目标场景恢复正常，边界与对照场景保持原有行为；既有测试文件和断言不作修改。
