package store

import (
	"context"
	"sync"
	"time"

	"github.com/w6xian/sidecar/internal/errors"
	"github.com/w6xian/sidecar/internal/muxhttp"

	"github.com/w6xian/sidecar/internal/config"

	"github.com/w6xian/sqlm"
	"go.uber.org/zap"
)

type TrackIdReq struct {
	TrackId string   `json:"track_id"`
	Tracker *Tracker `json:"-"`
}

// TrackIdReq 验证 validate
func (r *TrackIdReq) Validate() error {
	if r.TrackId == "" {
		return errors.New("track_id is required")
	}
	return nil
}

type AppIdReq struct {
	AppId   string   `json:"app_id"`
	Tracker *Tracker `json:"-"`
}

// AppIdReq 验证 validate
func (r *AppIdReq) Validate() error {
	if r.AppId == "" {
		return errors.NewL(r.Tracker, "app_id is required", "app_id is required")
	}
	return nil
}

type IdReq struct {
	Id      int64    `json:"id"`
	Tracker *Tracker `json:"-"`
}

// IdReq 验证 validate
func (r *IdReq) Validate() error {
	if r.Id == 0 {
		return errors.NewL(r.Tracker, "id is required", "id is required")
	}
	return nil
}

type IdTypeReq struct {
	Id      int64    `json:"id"`
	Type    string   `json:"type"`
	Tracker *Tracker `json:"-"`
}

// IdTypeReq 验证 validate
func (r *IdTypeReq) Validate() error {
	if r.Id == 0 {
		return errors.NewL(r.Tracker, "id is required", "id is required")
	}
	if r.Type == "" {
		r.Type = "cate"
	}
	return nil
}

type SnReq struct {
	Sn      string   `json:"sn"`
	Tracker *Tracker `json:"-"`
}

// SnReq 验证 validate
func (r *SnReq) Validate() error {
	if r.Sn == "" {
		return errors.NewL(r.Tracker, "sn is required", "sn is required")
	}
	return nil
}

type NameReq struct {
	Name    string   `json:"name"`
	Ts      int64    `json:"ts"`
	Tracker *Tracker `json:"-"`
}

// NameReq 验证 validate
func (r *NameReq) Validate() error {
	if r.Name == "" {
		return errors.NewL(r.Tracker, "name is required", "name is required")
	}
	return nil
}

type IdNameResp struct {
	Id   int64  `json:"id"`
	Name string `json:"name"`
}

type CodeReq struct {
	Code    string   `json:"code"`
	Ts      int64    `json:"ts"`
	Tracker *Tracker `json:"-"`
}

// CodeReq 验证 validate
func (r *CodeReq) Validate() error {
	if r.Code == "" {
		return errors.NewL(r.Tracker, "code is required", "code is required")
	}
	return nil
}

type OpenIdReq struct {
	Id      int64    `json:"id"`
	OpenId  string   `json:"open_id"`
	Tracker *Tracker `json:"-"`
}

// OpenIdReq 验证 validate
func (r *OpenIdReq) Validate() error {
	if r.Id == 0 {
		return errors.NewL(r.Tracker, "id is required", "id is required")
	}
	if r.OpenId == "" {
		return errors.NewL(r.Tracker, "open_id is required", "open_id is required")
	}
	return nil
}

type UnionIdReq struct {
	Id      int64    `json:"id"`
	UnionId string   `json:"union_id"`
	Tracker *Tracker `json:"-"`
}

// UnionIdReq 验证 validate
func (r *UnionIdReq) Validate() error {
	if r.Id == 0 {
		return errors.NewL(r.Tracker, "id is required", "id is required")
	}
	if r.UnionId == "" {
		return errors.NewL(r.Tracker, "union_id is required", "union_id is required")
	}
	return nil
}

type UnionIdsReq struct {
	Id       int64    `json:"id"`
	UnionIds []string `json:"union_ids"`
	Tracker  *Tracker `json:"-"`
}

// UnionIdsReq 验证 validate
func (r *UnionIdsReq) Validate() error {
	if r.Id == 0 {
		return errors.NewL(r.Tracker, "id is required", "id is required")
	}
	if len(r.UnionIds) == 0 {
		return errors.NewL(r.Tracker, "union_ids is required", "union_ids is required")
	}
	return nil
}

type CostReq struct {
	Id      int64    `json:"id"`
	SKUId   int64    `json:"sku_id"`
	Cost    int64    `json:"cost"`
	Tracker *Tracker `json:"-"`
}

// CostReq 验证 validate
func (r *CostReq) Validate() error {
	if r.Id == 0 {
		return errors.NewL(r.Tracker, "id is required", "id is required")
	}
	return nil
}

type OkResp struct {
	Status int64 `json:"id"`
}

var locker = &sync.Mutex{}

type Store struct {
	profile  *config.Profile
	driver   Driver
	Lager    *zap.Logger
	Language string
}

func New(driver Driver, opt *config.Profile, lager *zap.Logger) (*Store, error) {
	store := &Store{
		profile:  opt,
		driver:   driver,
		Lager:    lager,
		Language: "zh",
	}
	return store, nil
}

func (s *Store) GetDriver() Driver {
	return s.driver
}

func (s *Store) GetConnect(ctx context.Context) context.Context {
	return s.driver.GetConnect(ctx)
}

func (s *Store) CloseConnect(ctx context.Context) error {
	return s.driver.CloseConnect(ctx)
}

func (s *Store) Close() error {
	// Stop all cache cleanup goroutines
	return s.driver.Close()
}

func (s *Store) DbApiConnectWithClose(ctx context.Context) (context.Context, func()) {
	locker.Lock()
	defer locker.Unlock()
	reqId := time.Now().UnixNano()
	ctx = context.WithValue(ctx, muxhttp.ContextKey("request_id"), reqId)
	return s.driver.GetConnect(ctx), func() {
		s.driver.CloseConnect(ctx)
	}
}

func (s *Store) GetLink(ctx context.Context) sqlm.ITable {
	return s.driver.GetLink(ctx)
}

func (s *Store) Action(ctx context.Context, f func(tx sqlm.ITable, args ...any) (int64, error)) (int64, error) {
	link := s.driver.GetAction(ctx)
	defer link.Close()
	return link.Action(f)
}

// DbConnectWithClose 数据库连接上下文，使用后必须关闭
// 示例：
// ctx, close := s.DbConnectWithClose(context.Background())
// defer close()
// 需要在入口函数中调用，确保在请求结束后关闭数据库连接
func (s *Store) DbConnectWithClose(ctx context.Context) (context.Context, func()) {
	return s.driver.GetConnect(ctx), func() {
		s.driver.CloseConnect(ctx)
	}
}
