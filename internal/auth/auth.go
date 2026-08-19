// Package auth 提供 JWT 访问令牌签发与校验，以及密码哈希。
package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Claims 访问令牌声明。
type Claims struct {
	UserID   int64    `json:"uid"`
	Username string   `json:"usr"`
	Roles    []string `json:"rls"`
	Issuer   string   `json:"iss"`
	IssuedAt int64    `json:"iat"`
	ExpireAt int64    `json:"exp"`
}

// TokenService 令牌签发与校验。
type TokenService struct {
	secret []byte
	issuer string
	ttl    time.Duration
}

// NewTokenService 创建令牌服务。
func NewTokenService(secret, issuer string, ttl time.Duration) *TokenService {
	return &TokenService{secret: []byte(secret), issuer: issuer, ttl: ttl}
}

// Sign 签发访问令牌。
func (s *TokenService) Sign(claims Claims) (string, error) {
	claims.Issuer = s.issuer
	claims.IssuedAt = time.Now().Unix()
	claims.ExpireAt = time.Now().Add(s.ttl).Unix()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	body, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	payload := base64.RawURLEncoding.EncodeToString(body)
	signingInput := header + "." + payload
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(signingInput))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return signingInput + "." + sig, nil
}

// Verify 校验访问令牌并返回声明。
func (s *TokenService) Verify(token string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, errors.New("令牌格式无效")
	}
	signingInput := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(signingInput))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return Claims{}, errors.New("令牌签名无效")
	}
	body, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, err
	}
	var c Claims
	if err := json.Unmarshal(body, &c); err != nil {
		return Claims{}, err
	}
	if c.ExpireAt < time.Now().Unix() {
		return Claims{}, errors.New("令牌已过期")
	}
	if c.Issuer != s.issuer {
		return Claims{}, errors.New("令牌签发者不匹配")
	}
	return c, nil
}

// HashPassword 使用 bcrypt 哈希密码。
func HashPassword(plain string) (string, error) {
	if len(plain) < 8 {
		return "", fmt.Errorf("密码长度不能少于 8 位")
	}
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// VerifyPassword 校验密码。
func VerifyPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

// HashToken 对刷新令牌原文做 SHA-256 哈希后落库，避免明文存储。
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// RandomToken 生成高强度随机令牌原文。
func RandomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
