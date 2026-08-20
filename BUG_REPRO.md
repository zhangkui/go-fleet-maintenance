# BUG-027 Reproduction

## 现象
写入审计日志后，actor_username 期望是 admin，实际却变成了 172.29.0.1；IP 字段反而写成 admin，details 里也找不到传入的 balance。其他审计字段没有问题，麻烦帮我把这几个字段保存错位的问题修一下。

## 触发命令

    go test ./internal/service -count=1 -run '^TestBug027_AuditAppendFidelity$'

## 缺陷基线结果
固定回归命令在缺陷分支退出码为 1，目标业务断言失败。

## 失效链路
涉及生产文件：internal/service/audit.go、internal/repository/mysql/audit.go。请求输入或任务状态在这些职责层之间传递时被错误转换、遗漏或使用，最终形成题面中的可见结果。

## Gold 机制变化

    -	if detail == nil {
    +	if detail != nil {
    -		VALUES(?,?,?,?,?,?,?)`, a.ActorUserID, a.IP, a.Action, a.ResourceType, a.ResourceID, a.Detail, a.ActorName)
    +		VALUES(?,?,?,?,?,?,?)`, a.ActorUserID, a.ActorName, a.Action, a.ResourceType, a.ResourceID, a.Detail, a.IP)

## 正确结果
题面中的目标场景恢复正常，边界与对照场景保持原有行为；既有测试文件和断言不作修改。
