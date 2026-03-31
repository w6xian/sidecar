package store

import (
	"context"

	"github.com/w6xian/sqlm"
)

type Driver interface {
	// basic
	Close() error
	GetConnect(ctx context.Context) context.Context
	CloseConnect(ctx context.Context) error
	GetAction(ctx context.Context) *sqlm.Db
	GetLink(ctx context.Context) sqlm.ITable
	// GetToken gets token by app key.
	GetToken(link sqlm.ITable, appId string) (*Token, error)
}
