package entity

import "time"

// User 系统用户实体。
type User struct {
	ID               int64      `json:"id"`
	Username         string     `json:"username"`
	PasswordHash     string     `json:"-"`
	Email            string     `json:"email"`
	FullName         string     `json:"full_name"`
	Status           string     `json:"status"`
	FailedLoginCount int        `json:"-"`
	LockedUntil      *time.Time `json:"-"`
	LastLoginAt      *time.Time `json:"last_login_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// UserStatus 常量。
const (
	UserStatusActive   = "active"
	UserStatusDisabled = "disabled"
)

// RefreshToken 可撤销的刷新令牌，哈希后落库。
type RefreshToken struct {
	ID        int64
	UserID    int64
	TokenHash string
	IssuedAt  time.Time
	ExpiresAt time.Time
	RevokedAt *time.Time
	ClientIP  string
}

// Credentials 登录凭据。
type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// AuthTokens 登录成功后返回的令牌对。
type AuthTokens struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int64     `json:"expires_in"`
	ExpiresAt    time.Time `json:"expires_at"`
	User         User      `json:"user"`
	Permissions  []string  `json:"permissions"`
}

// PasswordChange 修改密码请求。
type PasswordChange struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}
