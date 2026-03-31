package mw

import (
	"context"
	"net/http"

	"github.com/w6xian/sidecar/internal/crypto/aes"
	"github.com/w6xian/sidecar/internal/muxhttp"
	"github.com/w6xian/sidecar/internal/options"
	"github.com/w6xian/sidecar/internal/utils"
)

func AESDecode(key string) muxhttp.Decorator {
	return func(f muxhttp.Handler) muxhttp.Handler {
		return func(w http.ResponseWriter, r *http.Request) ([]byte, error) {
			risk := options.RiskData{}
			err := utils.DecodeJSONBody(w, r, &risk)
			if err != nil {
				w.Write([]byte(err.Error()))
				r.Context().Done()
				return nil, nil
			}
			data, err := aes.Base64AESEBCDecrypt(risk.Body, []byte(key))
			if err != nil {
				w.Write([]byte(err.Error()))
				r.Context().Done()
				return nil, nil
			}
			// fmt.Println(string(data))
			// 更新Context
			r = r.WithContext(context.WithValue(r.Context(), muxhttp.ContextKey("aes_data"), data))
			return f(w, r)
		}
	}
}
