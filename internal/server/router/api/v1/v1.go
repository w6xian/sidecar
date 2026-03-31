package v1

import (
	"context"

	"github.com/w6xian/sidecar/internal/config"
	"github.com/w6xian/sidecar/internal/store"

	"go.uber.org/zap"
)

type Api struct {
	Context context.Context
	Store   *store.Store
	Profile *config.Profile
	Lager   *zap.Logger
}

func NewApi(ctx context.Context, store *store.Store, profile *config.Profile, lager *zap.Logger) *Api {
	return &Api{
		Context: ctx,
		Store:   store,
		Profile: profile,
		Lager:   lager,
	}
}

func (v *Api) DbConnectWithClose(ctx context.Context) (context.Context, func()) {
	return v.Store.GetConnect(ctx), func() {
		v.Store.CloseConnect(ctx)
	}
}
