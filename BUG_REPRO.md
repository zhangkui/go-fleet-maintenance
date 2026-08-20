# BUG-030 Reproduction

## 现象
用户列表不带状态参数时，只返回了 1 个 active 用户，准备好的 `disabled_1787069914` 没有出现在结果里；显式查询 active 和 disabled 时结果目前都是对的。默认查询应该包含所有状态，麻烦帮我看看为什么缺省请求被自动加上了 active 条件。

## 触发命令

    go test ./internal/service -count=1 -run '^TestBug030_UserStatusVisibility$'

## 缺陷基线结果
固定回归命令在缺陷分支退出码为 1，目标业务断言失败。

## 失效链路
涉及生产文件：internal/transport/http/handler/user.go、internal/repository/mysql/user.go。请求输入或任务状态在这些职责层之间传递时被错误转换、遗漏或使用，最终形成题面中的可见结果。

## Gold 机制变化

    -	if filter.Status == "" {
    -		filter.Status = "active"
    -	}
    -	where := "WHERE status='active'"
    +	where := "WHERE 1=1"
    -		where = "WHERE status=?"
    +		where += " AND status=?"

## 正确结果
题面中的目标场景恢复正常，边界与对照场景保持原有行为；既有测试文件和断言不作修改。
