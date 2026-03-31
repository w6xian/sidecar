package auth

import (
	"encoding/json"
	"fmt"

	"github.com/w6xian/sidecar/internal/utils"
)

const (
	// 收银员
	CasherTypeCasher = 1
	// 收银员助手
	CasherTypeCasherHelper = 2
	// 收银员助手
	CasherTypeAdmin = 3
)

type CasherJwtClaims struct {
	Claims   *CasherClaims
	AppId    string
	JwtToken string
}

type CasherClaims struct {
	TrackID    int64  `json:"track_id"`
	SessionId  int64  `json:"session_id"`
	AppId      string `json:"app_id"`
	ShopId     int64  `json:"shop_id"`
	ProxyId    int64  `json:"proxy_id"`
	UserId     int64  `json:"user_id"`
	CasherId   int64  `json:"casher_id"`
	EmployeeId int64  `json:"employee_id"`
	Name       string `json:"name"`
}

func (u *CasherClaims) Unmarshal(b []byte) error {
	js := utils.JsonValue{}
	err := json.Unmarshal(b, &js)
	if err != nil {
		return err
	}
	u.AppId = js.String("app_id")
	u.ProxyId = js.Int64("proxy_id")
	u.CasherId = js.Int64("casher_id")
	u.TrackID = js.Int64("track_id")
	u.SessionId = js.Int64("session_id")
	u.ShopId = js.Int64("shop_id")
	u.UserId = js.Int64("user_id")
	u.Name = js.String("name")
	u.EmployeeId = js.Int64("employee_id")

	return nil
}

func NewCasherClaims(c map[string]any) *CasherClaims {

	jsonStr, err := json.Marshal(c)
	if err != nil {
		return nil
	}
	cc := &CasherClaims{}
	fmt.Println(string(jsonStr))
	err = cc.Unmarshal(jsonStr)
	if err != nil {
		return nil
	}
	return cc
}
