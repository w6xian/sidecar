package auth

import (
	"context"
	"errors"
	"sync"

	"github.com/w6xian/sidecar/internal/store"
	"go.uber.org/zap"
)

var (
	ErrInvalidToken = errors.New("invalid or expired token")
	ErrEmptyToken   = errors.New("token is empty")
)

// Authenticator 鉴权接口
type Authenticator interface {
	Validate(appId string, sign string, t int64) (*store.Token, error)
}

// TokenAuth 基于静态 Token 的鉴权实现
type TokenAuth struct {
	mu      sync.RWMutex
	Store   *store.Store
	enabled bool
	tokens  map[string]Token
}

type Token struct {
	Token  string `json:"token"`
	AppKey string `json:"app_key"`
	AppSec string `json:"app_sec"`
	Alias  string `json:"alias"`
}

// NewTokenAuth 创建 Token 鉴权器
func NewTokenAuth(logger *zap.Logger, store *store.Store) *TokenAuth {
	t := &TokenAuth{
		Store:   store,
		enabled: false,
	}
	return t
}

// Validate 验证 Token
func (a *TokenAuth) Validate(appId string, sign string, t int64) (*store.Token, error) {
	if appId == "" {
		return nil, ErrEmptyToken
	}
	if sign == "" {
		return nil, ErrEmptyToken
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	ctx := context.Background()
	// 1 初始化数据库连接
	ctx, close := a.Store.DbConnectWithClose(ctx)
	defer close()
	token, err := a.Store.GetToken(ctx, appId, sign, t)
	if token == nil {
		return nil, ErrInvalidToken
	}
	if err != nil {
		return nil, err
	}
	return token, nil
}

// IPWhitelist IP 白名单验证
type IPWhitelist struct {
	mu        sync.RWMutex
	whitelist map[string]bool
	enabled   bool
}

// NewIPWhitelist 创建 IP 白名单
func NewIPWhitelist(ips []string) *IPWhitelist {
	w := &IPWhitelist{
		whitelist: make(map[string]bool, len(ips)),
		enabled:   len(ips) > 0,
	}
	for _, ip := range ips {
		w.whitelist[ip] = true
	}
	return w
}

// IsAllowed 检查 IP 是否在白名单中
func (w *IPWhitelist) IsAllowed(ip string) bool {
	if !w.enabled {
		return true // 未启用白名单，允许所有
	}
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.whitelist[ip]
}
