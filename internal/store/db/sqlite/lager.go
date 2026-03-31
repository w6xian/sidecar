package sqlite

import (
	"github.com/w6xian/sqlm"
)

// 用于拿日志
func (s *DB) GetMap(link sqlm.ITable, tableName string, pk string, value string) map[string]any {
	row, err := link.Table(tableName).
		Where("%s='%s'", pk, value).
		Query()
	if err != nil {
		return map[string]any{}
	}
	return row.ToMap()
}

func (s *DB) GetMapById(link sqlm.ITable, tableName string, id int64) map[string]any {
	row, err := link.Table(tableName).
		Where("id=%d", id).
		Query()
	if err != nil {
		return map[string]any{}
	}
	return row.ToMap()
}

// 用于拿日志
func (s *DB) SelectMap(link sqlm.ITable, tableName string, pk string, value string) []map[string]any {
	rows, err := link.Table(tableName).
		Where("%s='%s'", pk, value).
		QueryMulti()
	if err != nil {
		return []map[string]any{}
	}
	return rows.ToArray()
}
