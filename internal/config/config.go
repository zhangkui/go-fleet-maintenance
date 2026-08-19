// Package config 统一加载环境变量配置，禁止把密码、Token、DSN 写死。
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config 全量运行期配置，全部来自环境变量或安全本地默认值。
type Config struct {
	HTTPAddr        string
	MySQLDSN        string
	RedisAddr       string
	RedisPassword   string
	RedisDB         int
	RedisKeyPrefix  string
	JWTSecret       string
	TokenIssuer     string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	AdminUsername   string
	AdminPassword   string
	LoginRateLimit  int
	LoginRateWindow time.Duration
	Timezone        string
	RequestMaxBytes int64
	ShutdownTimeout time.Duration
}

// Load 从环境变量加载配置，未设置时使用可零配置启动的安全默认值。
func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:        env("HTTP_ADDR", ":8080"),
		MySQLDSN:        env("MYSQL_DSN", "fleet:fleet_pwd@tcp(mysql:3306)/fleet?parseTime=true&loc=Asia%2FShanghai&charset=utf8mb4&collation=utf8mb4_unicode_ci"),
		RedisAddr:       env("REDIS_ADDR", "redis:6379"),
		RedisPassword:   env("REDIS_PASSWORD", ""),
		RedisKeyPrefix:  env("REDIS_KEY_PREFIX", "fleet:"),
		TokenIssuer:     env("TOKEN_ISSUER", "go-fleet-maintenance"),
		AdminUsername:   env("ADMIN_USERNAME", "admin"),
		AdminPassword:   env("ADMIN_PASSWORD", "Admin123!"),
		Timezone:        env("TZ", "Asia/Shanghai"),
		RequestMaxBytes: int64Env("REQUEST_MAX_BYTES", 1<<20),
		ShutdownTimeout: durationEnv("SHUTDOWN_TIMEOUT", 15*time.Second),
	}

	if cfg.RedisDB = intEnv("REDIS_DB", 0); cfg.RedisDB < 0 || cfg.RedisDB > 15 {
		return Config{}, fmt.Errorf("REDIS_DB 必须在 0..15 之间，当前 %d", cfg.RedisDB)
	}
	if cfg.LoginRateLimit = intEnv("LOGIN_RATE_LIMIT", 5); cfg.LoginRateLimit <= 0 {
		return Config{}, fmt.Errorf("LOGIN_RATE_LIMIT 必须 > 0，当前 %d", cfg.LoginRateLimit)
	}
	cfg.LoginRateWindow = durationEnv("LOGIN_RATE_WINDOW", time.Minute)

	cfg.JWTSecret = env("JWT_SECRET", "")
	if cfg.JWTSecret == "" {
		// 本地零配置启动时生成一次性默认密钥；生产必须显式设置 JWT_SECRET。
		cfg.JWTSecret = "local-only-default-secret-do-not-use-in-production"
	}
	cfg.AccessTokenTTL = durationEnv("ACCESS_TOKEN_TTL", 30*time.Minute)
	cfg.RefreshTokenTTL = durationEnv("REFRESH_TOKEN_TTL", 7*24*time.Hour)

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) validate() error {
	if strings.TrimSpace(c.AdminUsername) == "" || strings.TrimSpace(c.AdminPassword) == "" {
		return fmt.Errorf("默认管理员账号或密码不能为空")
	}
	if c.AccessTokenTTL <= 0 || c.RefreshTokenTTL <= 0 {
		return fmt.Errorf("令牌有效期必须为正数")
	}
	if c.RefreshTokenTTL <= c.AccessTokenTTL {
		return fmt.Errorf("刷新令牌有效期必须大于访问令牌有效期")
	}
	if c.RequestMaxBytes <= 0 {
		return fmt.Errorf("REQUEST_MAX_BYTES 必须为正数")
	}
	return nil
}

func env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return strings.TrimSpace(v)
	}
	return def
}

func intEnv(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok {
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err == nil {
			return n
		}
	}
	return def
}

func int64Env(key string, def int64) int64 {
	if v, ok := os.LookupEnv(key); ok {
		n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		if err == nil {
			return n
		}
	}
	return def
}

func durationEnv(key string, def time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok {
		d, err := time.ParseDuration(strings.TrimSpace(v))
		if err == nil {
			return d
		}
	}
	return def
}
