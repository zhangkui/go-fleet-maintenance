# BUG-028 Reproduction

## 现象
审计日志按 action 查询时，传一个不存在的 action，结果 total 仍然是 1，预期应该为 0；查询 auth.login 和按 resource 过滤时目前还能得到正确结果。不要直接修复，只把分析结果给我；帮我找出 action 条件为何没有真正限制结果，并说明它和 resource 条件一起使用时会发生什么。

## 触发命令

    go test ./internal/service -count=1 -run '^TestBug028_AuditActionFilter$'

## 缺陷基线结果
固定回归命令在缺陷分支退出码为 1，目标业务断言失败。

## 失效链路
涉及生产文件：internal/transport/http/handler/user.go、internal/repository/mysql/audit.go。请求输入或任务状态在这些职责层之间传递时被错误转换、遗漏或使用，最终形成题面中的可见结果。

## Gold 机制变化

    -	filter.Status = r.URL.Query().Get("resource_type")
    +	filter.Status = r.URL.Query().Get("action")
    -		where += " AND resource_type=?"
    +		where += " AND action=?"

## 正确结果
题面中的目标场景恢复正常，边界与对照场景保持原有行为；既有测试文件和断言不作修改。
