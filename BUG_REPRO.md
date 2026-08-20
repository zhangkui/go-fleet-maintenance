# BUG-024 Reproduction

## 现象
提醒扫描接口传 `days=60` 时，本应扫到目标提醒，实际返回数量是 0；不传 days 和传非法值时使用的范围也需要确认。项目内容保持原样，只需要调查结果；帮我确认请求里的天数到底有没有参与截止日期计算，并把三种输入对应的实际范围说明清楚。

## 触发命令

    go test ./internal/service -count=1 -run '^TestBug024_ReminderScanHorizon$'

## 缺陷基线结果
固定回归命令在缺陷分支退出码为 1，目标业务断言失败。

## 失效链路
涉及生产文件：internal/transport/http/handler/reminder.go、internal/service/reminder.go。请求输入或任务状态在这些职责层之间传递时被错误转换、遗漏或使用，最终形成题面中的可见结果。

## Gold 机制变化

    -	lookahead, _ := strconv.Atoi(r.URL.Query().Get("lookahead"))
    +	lookahead, _ := strconv.Atoi(r.URL.Query().Get("days"))
    -		lookaheadDays = 365
    +		lookaheadDays = 30

## 正确结果
题面中的目标场景恢复正常，边界与对照场景保持原有行为；既有测试文件和断言不作修改。
