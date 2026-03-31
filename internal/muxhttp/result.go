package muxhttp

import (
	"encoding/json"
)

type WithTotal struct {
	Total int64 `json:"total,omitempty"`
}

type WidthPragma struct {
	Response V2Header `json:"response"`
}

type Result struct {
	Status int               `json:"status"`
	Text   string            `json:"text"`
	Track  string            `json:"track"`
	Data   interface{}       `json:"data"`
	Extra  map[string]string `json:"extra,omitempty"`
}

func (r *Result) ToBytes() ([]byte, error) {
	res, err := json.Marshal(r)
	return res, err
}

func (r *Result) Error() string {
	return r.Text
}
func (r *Result) MessageId() string {
	return r.Track
}

type RowsResult struct {
	Result
	WithTotal
}

type PragmaResult struct {
	Result
	WidthPragma
}
