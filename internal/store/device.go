package store

import (
	"context"

	"github.com/w6xian/sidecar/internal/auth"
	"github.com/w6xian/sidecar/internal/i18n"

	"golang.org/x/text/language"
)

type Device struct {
	ShopId    int64  `json:"shop_id"`
	MachineId string `json:"machine_id"`
}

type Tracker struct {
	ProxyId int64 `json:"proxy_id"`
	// ShopId 店铺Id
	ShopId    int64  `json:"shop_id"`
	RoomId    int64  `json:"room_id"`
	ShopName  string `json:"shop_name"`
	ComId     int64  `json:"com_id"`
	StoreId   int64  `json:"store_id"`
	MachineId string `json:"machine_id"`
	AppId     string `json:"app_id"`
	OpenId    string `json:"open_id"`
	// 序号
	MachineNo   int64  `json:"machine_no"`
	TrackId     string `json:"track_id"`
	HandlerId   int64  `json:"handler_id"`
	HandlerName string `json:"handler_name"`
	Language    string `json:"language"`
	JwtToken    string `json:"jwt_token"`
}

func (t *Tracker) L(key string, def string, fields ...i18n.Field) string {
	if t.Language == "" {
		t.Language = language.Chinese.String()
	}
	l := len(fields)
	if l == 0 {
		return i18n.T(t.Language, key, def)
	}

	data := i18n.D{}
	for _, f := range fields {
		data[f.Key] = f.Value()
	}
	return i18n.TWithData(t.Language, key, def, data)
}

func (t *Tracker) AuthHeader() *Header {
	return &Header{
		TrackId: t.TrackId,
		AppId:   t.AppId,
		Lang:    t.Language,
		Auth:    t.JwtToken,
	}
}

func NewAnonimousTracker(ctx context.Context) *Tracker {
	return &Tracker{
		MachineNo: 1,
	}
}

func NewTracker(claims *auth.CasherClaims) *Tracker {
	return &Tracker{
		ProxyId:     claims.ProxyId,
		ShopId:      claims.ShopId,
		AppId:       claims.AppId,
		HandlerId:   claims.CasherId,
		HandlerName: claims.Name,
		MachineNo:   1,
	}
}

type Handler struct {
	ShopId      int64  `json:"shop_id"`
	HandlerId   int64  `json:"handler_id"`
	HandlerName string `json:"handler_name"`
}
