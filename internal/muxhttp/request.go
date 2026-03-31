package muxhttp

import (
	"encoding/json"
	"net/http"

	ilog "github.com/w6xian/sidecar/internal/logger"
	"github.com/w6xian/sidecar/internal/utils"
)

func GetRequestId(req *http.Request) string {
	reqId := req.Context().Value(ContextKey("request_id")).(int64)
	return utils.GetString(reqId)
}

func GetLanguage(req *http.Request) string {
	lang := req.Context().Value(ContextKey("language")).(string)
	return lang
}

func GetLogger(req *http.Request) ilog.StdLog {
	log := req.Context().Value(ContextKey("logger")).(ilog.StdLog)
	return log
}

type HttpPostJson struct {
	Data []byte
}

func (h *HttpPostJson) Unmarshal(v any) error {
	// fmt.Println(string(h.Data))
	return json.Unmarshal(h.Data, v)
}

func (h *HttpPostJson) Maps() map[string]any {
	v := make(map[string]any)
	json.Unmarshal(h.Data, &v)
	return v
}

type HttpPostJsonData []byte

func (h HttpPostJsonData) Unmarshal(v any) error {
	// fmt.Println(string(h))
	return json.Unmarshal(h, v)
}

func (h HttpPostJsonData) Maps() map[string]any {
	v := make(map[string]any)
	json.Unmarshal(h, &v)
	return v
}
