package store

import (
	"context"
	"fmt"
	"time"

	"github.com/w6xian/sidecar/internal/crypto/aes"
	"github.com/w6xian/sidecar/internal/errors"
)

type Token struct {
	Token      string `json:"token"`
	AppId      string `json:"app_id"`
	AppSec     string `json:"app_sec"`
	AppSn      string `json:"app_sn"`
	Name       string `json:"name"`
	Intime     int64  `json:"intime"`
	ExpireTime int64  `json:"expire_time"`
	LockStatus int    `json:"lock_status"`
	LockTime   int64  `json:"lock_time"`
}

// ErrTokenExpired 令牌过期错误
var ErrTokenExpired = errors.New("token expired")

// ErrTokenLocked 令牌被锁定错误
var ErrTokenLocked = errors.New("token locked")

// ErrInvalidToken 无效令牌错误
var ErrInvalidToken = errors.New("invalid token")

func (s *Store) GetToken(ctx context.Context, appId string, sign string, t int64) (*Token, error) {
	link := s.GetLink(ctx)
	token, err := s.GetDriver().GetToken(link, appId)
	if err != nil {
		return nil, err
	}
	// 1 校验 token 是否过期
	if token.ExpireTime > 0 && token.ExpireTime < time.Now().Unix() {
		return nil, ErrTokenExpired
	}
	// 2 校验 token 是否被锁定
	if token.LockStatus == 1 {
		return nil, ErrTokenLocked
	}
	// 3 校验 appSec 是否匹配
	appSec := fmt.Sprintf("%d-%s-%d", t, token.AppSec, t)
	// appSec 加密后与 sign 对比
	appSec, _ = aes.Base64AESEBCEncrypt([]byte(appSec), aes.GetAES256Key([]byte(token.AppSn)))
	if appSec != sign {
		return nil, ErrInvalidToken
	}
	return token, nil
}
