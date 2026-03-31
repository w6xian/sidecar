package sqlite

import (
	"github.com/w6xian/sidecar/internal/store"

	"github.com/w6xian/sqlm"
)

func (s *DB) GetToken(link sqlm.ITable, appKey string) (*store.Token, error) {
	row, err := link.Table(store.TABLE_SIDECAR_TOKENS).
		Where("app_id='%s'", appKey).
		Query()
	if err != nil {
		return nil, err
	}
	token := &store.Token{}
	row.Scan(token)

	return token, nil

}
