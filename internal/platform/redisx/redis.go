// Package redisx 提供 Redis 客户端、缓存、幂等标记与限流封装。
package redisx

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client 封装 go-redis 客户端与统一 key 前缀。
type Client struct {
	rdb       *redis.Client
	keyPrefix string
}

// New 创建 Redis 客户端。
func New(addr, password string, db int, keyPrefix string) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		DialTimeout:  3 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
		PoolSize:     16,
	})
	return &Client{rdb: rdb, keyPrefix: keyPrefix}, nil
}

// Ping 等待 Redis 就绪。
func (c *Client) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return c.rdb.Ping(ctx).Err()
}

// Rdb 返回底层客户端，供高级场景使用。
func (c *Client) Rdb() *redis.Client { return c.rdb }

// key 拼接完整 key。
func (c *Client) key(k string) string { return c.keyPrefix + k }

// SetCache 写入热点查询缓存。
func (c *Client) SetCache(ctx context.Context, key string, val []byte, ttl time.Duration) error {
	if c == nil || c.rdb == nil {
		return ErrRedisUnavailable
	}
	if ttl == 0 {
		ttl = 24 * time.Hour
	}
	return c.rdb.Set(ctx, c.key("cache:"+key), val, ttl).Err()
}

// GetCache 读取热点查询缓存。
func (c *Client) GetCache(ctx context.Context, key string) ([]byte, error) {
	if c == nil || c.rdb == nil {
		return nil, ErrRedisUnavailable
	}
	v, err := c.rdb.Get(ctx, c.key("cache:"+key)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	return v, err
}

// InvalidateCache 删除缓存 key。
func (c *Client) InvalidateCache(ctx context.Context, keys ...string) error {
	if c == nil || c.rdb == nil {
		return ErrRedisUnavailable
	}
	if len(keys) == 0 {
		return nil
	}
	full := make([]string, len(keys))
	for i, k := range keys {
		full[i] = c.key("cache:" + k)
	}
	return c.rdb.Del(ctx, full...).Err()
}

// SetIdempotency 设置短期幂等标记，返回是否首次写入。
// 若 key 已存在且 value 一致，返回 false=nil 表示重复请求；不一致返回 ErrIdempotentConflict。
func (c *Client) SetIdempotency(ctx context.Context, key, payload string, ttl time.Duration) (bool, error) {
	if c == nil || c.rdb == nil {
		// Redis 故障时降级：允许继续执行，幂等性退化为数据库唯一约束兜底。
		return true, nil
	}
	k := c.key("idem:" + key)
	ok, err := c.rdb.SetNX(ctx, k, payload, ttl).Result()
	if err != nil {
		return true, nil // 故障降级，依赖数据库唯一约束。
	}
	if !ok {
		existing, gerr := c.rdb.Get(ctx, k).Result()
		if gerr != nil {
			return false, nil
		}
		if existing != payload {
			return false, ErrIdempotentConflict
		}
		return false, nil
	}
	return true, nil
}

// AcquireLock 获取短时锁（用于避免并发重复触发维保计划生成等）。
func (c *Client) AcquireLock(ctx context.Context, key, owner string, ttl time.Duration) (bool, error) {
	if c == nil || c.rdb == nil {
		return true, nil // 降级：不强制加锁，依赖数据库约束保证一致。
	}
	ok, err := c.rdb.SetNX(ctx, c.key("lock:"+key), owner, ttl).Result()
	if err != nil {
		return true, nil
	}
	return ok, nil
}

// ReleaseLock 释放短时锁，仅持有者可释放。
func (c *Client) ReleaseLock(ctx context.Context, key, owner string) error {
	if c == nil || c.rdb == nil {
		return nil
	}
	k := c.key("lock:" + key)
	v, err := c.rdb.Get(ctx, k).Result()
	if err != nil {
		return nil
	}
	if v != owner {
		return nil
	}
	return c.rdb.Del(ctx, k).Err()
}

// HitRateLimit 计数限流，返回当前窗口内累计次数与是否超限。
func (c *Client) HitRateLimit(ctx context.Context, key string, window time.Duration, max int) (int, bool, error) {
	if c == nil || c.rdb == nil {
		// 降级：不强制限流，由数据库与审计兜底，但记录为未超限。
		return 0, false, nil
	}
	k := c.key("rl:" + key)
	pipe := c.rdb.TxPipeline()
	incr := pipe.Incr(ctx, k)
	pipe.Expire(ctx, k, window)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, false, nil
	}
	count := int(incr.Val())
	return count, count > max, nil
}

// Errors
var (
	ErrRedisUnavailable   = errors.New("redis 暂不可用")
	ErrIdempotentConflict = errors.New("幂等键重复且请求不一致")
)
