package jwt

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/w6xian/sidecar/internal/auth"
	"github.com/w6xian/sidecar/internal/auth/jwt/ED25519"
	"github.com/w6xian/sidecar/internal/timex"
	"github.com/w6xian/sidecar/internal/utils/id"

	"github.com/golang-jwt/jwt/v5"
)

const (
	ExpireTime = time.Hour * 24 * 7
	Issuer     = "51d.	ink"
)

func EncodeToken(claims *auth.CasherClaims) (string, error) {
	if claims == nil {
		return "", errors.New("claims is nil")
	}

	// 生成JWT令牌
	k := ED25519.FromPemKey("./private.pem")
	// 生成JWT令牌
	token := jwt.New(jwt.SigningMethodEdDSA)
	payload := token.Claims.(jwt.MapClaims)
	/**
		iss (issuer): 发行者
	    exp (expiration time): 过期时间
	    sub (subject): 主题
	    aud (audience): 接收者
	    iat (issued at): 发行时间
	    nbf (not before): 生效时间
	*/
	payload["sub"] = "casher"
	payload["token"] = id.GetUuid()
	payload["iat"] = timex.UnixTime()
	payload["exp"] = time.Now().Add(ExpireTime).Unix()
	payload["iss"] = Issuer
	payload["casher_id"] = claims.CasherId
	payload["app_id"] = claims.AppId
	payload["proxy_id"] = claims.ProxyId
	payload["shop_id"] = claims.ShopId
	payload["user_id"] = claims.UserId
	payload["employee_id"] = claims.EmployeeId
	payload["name"] = claims.Name

	// 对JWT令牌进行签名
	signedToken, err := token.SignedString(k.PrivateKey)
	if err != nil {
		return "", err
	}
	return signedToken, nil
}

func DecodeToken(signedToken string) (*auth.CasherClaims, error) {
	if signedToken == "" {
		return nil, errors.New("signedToken is empty")
	}
	publicKey, err := ED25519.DecodePublicPemKey("./public.pem")
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}
	// 验证JWT令牌的有效性
	parsedToken, err := jwt.Parse(signedToken, func(token *jwt.Token) (any, error) {
		return publicKey, nil
	})
	if err != nil {
		fmt.Println(err.Error())
	}

	if claims, ok := parsedToken.Claims.(jwt.MapClaims); ok && parsedToken.Valid {
		cc := auth.NewCasherClaims(claims)
		return cc, nil

	}
	return nil, errors.New("signedToken is empty")
}

func FromHeader(req *http.Request) (string, error) {
	token := req.Header.Get("Authorization")
	if token == "" {
		return "", errors.New("token is empty")
	}
	token = strings.TrimPrefix(token, "Bearer ")
	return token, nil
}

func FromHeaderWithClaims(req *http.Request) (*auth.CasherJwtClaims, error) {
	token := req.Header.Get("Authorization")
	if token == "" {
		return nil, errors.New("token is empty")
	}
	token = strings.TrimPrefix(token, "Bearer ")
	cc, err := DecodeToken(token)
	if err != nil {
		return nil, err
	}
	return &auth.CasherJwtClaims{
		Claims:   cc,
		JwtToken: token,
	}, nil
}

func TokenFromHeader(req *http.Request) string {
	token := req.Header.Get("Authorization")
	if token == "" {
		return ""
	}
	token = strings.TrimPrefix(token, "Bearer ")
	return token
}
