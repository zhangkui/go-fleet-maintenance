# BUG-025 Reproduction

## 现象
车队总览缓存设置后，TTL 变成了 86400 秒，设计上只应该保留一分钟左右；另外传入零 TTL 时，也不能悄悄变成永久缓存。缓存命中和未命中时的数据结果目前是正常的，麻烦帮我把过期时间处理修正确。

## 触发命令

    go test ./internal/service -count=1 -run '^TestBug025_FleetSummaryCacheTTL$'

## 缺陷基线结果
固定回归命令在缺陷分支退出码为 1，目标业务断言失败。

## 失效链路
涉及生产文件：internal/service/report.go、internal/platform/redisx/redis.go。请求输入或任务状态在这些职责层之间传递时被错误转换、遗漏或使用，最终形成题面中的可见结果。

## Gold 机制变化

    -		_ = s.redis.SetCache(ctx, cacheKey, b, 0)
    +		_ = s.redis.SetCache(ctx, cacheKey, b, 60*time.Second)
    -	if ttl == 0 {
    -		ttl = 24 * time.Hour
    -	}

## 正确结果
题面中的目标场景恢复正常，边界与对照场景保持原有行为；既有测试文件和断言不作修改。
