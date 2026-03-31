package mw

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/w6xian/sidecar/internal/utils"

	"github.com/gorilla/mux"
)

func CORSMethodMiddleware(allows []string) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := w.Header()
			rHeader := r.Header
			origin := rHeader.Get("Origin")
			pos := utils.Find(origin, allows)
			if pos < 0 && len(origin) > 0 {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			header.Set("Access-Control-Allow-Methods", header.Get("Allow"))
			header.Set("Access-Control-Allow-Origin", rHeader.Get("Origin"))
			header.Set("Access-Control-Allow-Credentials", "true")
			header.Set("Access-Control-Allow-Headers", "Origin, Cookie, X-File-Name, X-Requested-With, Content-Type, Accept, Orz-Auth3, Orz-API-Version, Orz-APP-Id, Authorization, Wechatpay-Serial, Wechatpay-Timestamp, Wechatpay-Nonce, Wechatpay-Signature, Node")
			header.Set("P3P", "CP=CAO PSA OUR")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func RespondV1(w http.ResponseWriter, code int, data interface{}) {
	var response []byte
	var err error
	var isJSON bool

	if code == 200 {
		switch data.(type) {
		case string:
			response = []byte(data.(string))
		case []byte:
			response = data.([]byte)
		case nil:
			response = []byte{}
		default:
			isJSON = true
			response, err = json.Marshal(data)
			if err != nil {
				data = err
			}
		}
	}

	if code != 200 {
		isJSON = true
		response, _ = json.Marshal(struct {
			Message string `json:"text"`
			Status  int    `json:"status"`
		}{
			Message: fmt.Sprintf("%s", data),
			Status:  code,
		})
	}

	if isJSON {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	}
	w.Header().Set("X-NSQ-Content-Type", "nsq; version=1.0")
	w.WriteHeader(code)
	w.Write(response)
}
