package muxhttp

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

const (
	SUCESS = 200
)

type KV map[string]any

type Ok struct {
	Result
}

type Response interface {
	ToBytes() ([]byte, error)
}

func NewOkStatus(text string) *Ok {
	er := &Ok{}
	er.Status = 200
	er.Text = text
	er.Track = uuid.NewString()
	return er
}

type RowData struct {
	Data map[string]any
}

func (r *RowData) ToBytes() ([]byte, error) {
	data := r.Data
	rst := &Result{}
	if data == nil {
		return NewError("数据采集错误").ToBytes()
	}
	rst.Status = http.StatusOK
	rst.Text = "操作成功"
	rst.Track = uuid.NewString()
	rst.Data = data
	res, err := json.Marshal(rst)
	return res, err
}

type PragmaData struct {
	Data     KV
	Response V2Header
}

type V2Header struct {
	Headers map[string]string `json:"headers"`
}

func (r *PragmaData) ToBytes() ([]byte, error) {
	data := r.Data
	rst := &PragmaResult{}
	if data == nil {
		return NewError("数据采集错误").ToBytes()
	}
	rst.Status = http.StatusOK
	rst.Text = "操作成功"
	rst.Track = uuid.NewString()
	rst.Data = data
	rst.Response = r.Response
	res, err := json.Marshal(rst)
	return res, err
}

func (r *PragmaData) WithParam(pragmaCache string) *PragmaData {
	hs := V2Header{}
	h := map[string]string{}
	h["pragma"] = pragmaCache
	hs.Headers = h
	r.Response = hs
	return r
}

type RowsData struct {
	Data  []map[string]any
	Total int64
}

func (r *RowsData) ToBytes() ([]byte, error) {
	data := r.Data
	rst := &RowsResult{}
	if data == nil {
		return NewError("数据采集错误").ToBytes()
	}
	rst.Status = http.StatusOK
	rst.Text = "操作成功"
	rst.Track = uuid.NewString()
	rst.Data = data
	rst.Total = r.Total
	rst.Extra = map[string]string{}
	res, err := json.Marshal(rst)
	return res, err
}

func NewRowsData(data []map[string]any, total int64) Response {
	rd := &RowsData{
		Data:  data,
		Total: total,
	}
	return rd
}

func NewRowData(data map[string]any) Response {
	rd := &RowData{
		Data: data,
	}
	return rd
}

func NewMapData(data map[string]any) Response {
	rd := &RowData{
		Data: data,
	}
	return rd
}

func NewPragmaData(data map[string]any) *PragmaData {
	rd := &PragmaData{
		Data: data,
	}
	return rd
}

// 脱敏数据

type IMasking interface {
	Masking() any
}

type RiskData struct {
	Data any
}

func (r *RiskData) Filter(f ...func(any) any) *RiskData {
	if r.Data == nil {
		return r
	}
	for _, fn := range f {
		r.Data = fn(r.Data)
	}
	if len(f) == 0 {
		r.Data = FilterStructWithTag(r.Data)
	}

	return r
}

func (r *RiskData) ToBytes() ([]byte, error) {
	data := r.Data
	// 是不是实现了IMasking接口
	if masking, ok := data.(IMasking); ok {
		data = masking.Masking()
	}
	rst := &Result{}
	if data == nil {
		return NewError("数据采集错误").ToBytes()
	}
	rst.Status = http.StatusOK
	rst.Text = "操作成功"
	rst.Track = uuid.NewString()
	rst.Data = data
	res, err := json.Marshal(rst)
	return res, err
}

type RisksData struct {
	Data  any
	Total int64
	Extra map[string]string
}

func (r *RisksData) ToBytes() ([]byte, error) {
	data := r.Data
	// 是不是实现了IMasking接口
	if masking, ok := data.(IMasking); ok {
		data = masking.Masking()
	}
	rst := &RowsResult{}
	if data == nil {
		return NewError("数据采集错误").ToBytes()
	}
	rst.Status = http.StatusOK
	rst.Text = "操作成功"
	rst.Track = uuid.NewString()
	rst.Data = data
	rst.Total = r.Total
	if r.Extra != nil {
		rst.Extra = r.Extra
	}
	res, err := json.Marshal(rst)
	return res, err
}

func NewRiskData(data any) *RiskData {
	rd := &RiskData{
		Data: data,
	}
	return rd
}

func NewRisksData(data any, total int64) *RisksData {
	rd := &RisksData{
		Data:  data,
		Total: total,
	}
	return rd
}

func NewRisksWithExtra(data any, total int64, extra map[string]string) *RisksData {
	rd := &RisksData{
		Data:  data,
		Total: total,
		Extra: extra,
	}
	return rd
}

func ResponseV1Factory(f Handler) Handler {
	return func(w http.ResponseWriter, req *http.Request) ([]byte, error) {
		GetRequestId(req)
		r, e := f(w, req)
		// fmt.Println(reqId)
		if e != nil {
			ne := NewErr(e)
			er, _ := ne.ToBytes()
			w.WriteHeader(ne.Status)
			w.Write(er)
		} else {
			w.Write(r)
		}
		return []byte{}, nil
	}
}

func V2Response(f Handler) Handler {
	return func(w http.ResponseWriter, req *http.Request) ([]byte, error) {
		r, e := f(w, req)
		if e != nil {
			ne := NewErr(e)
			er, _ := ne.ToBytes()
			w.WriteHeader(http.StatusOK)
			w.Write(er)
		} else {
			w.Write(r)
		}
		return []byte{}, nil
	}
}
func ExcelResponse(f Handler) Handler {
	return func(w http.ResponseWriter, req *http.Request) ([]byte, error) {
		_, e := f(w, req)
		if e != nil {
			ne := NewErr(e)
			er, _ := ne.ToBytes()
			w.WriteHeader(http.StatusOK)
			w.Write(er)
		} else {
			w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
			w.Header().Set("Content-Disposition", "attachment; filename="+time.Now().Format("20060102")+".xlsx")
		}
		return []byte{}, nil
	}
}
