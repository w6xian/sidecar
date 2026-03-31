package mw

import (
	"context"
	"net/http"

	ilog "github.com/w6xian/sidecar/internal/logger"

	"github.com/gorilla/mux"
)

func Logger(log ilog.StdLog) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), "logger", log)
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}
