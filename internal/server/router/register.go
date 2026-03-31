package router

import (
	"context"
	"net/http"

	"github.com/w6xian/sidecar/internal/muxhttp"
	v1 "github.com/w6xian/sidecar/internal/server/router/api/v1"

	"github.com/gorilla/mux"
)

func Register(ctx context.Context, r *mux.Router, v *v1.Api) {
	// 收银相关
	register_api(r, "/", JsonV2(func(w http.ResponseWriter, req *http.Request) ([]byte, error) {
		return []byte("hello this is casher server"), nil
	})).Methods(http.MethodGet, http.MethodOptions)

	// 设置 setting
	// ### 同步操作

}

var apiRoutes map[string]string = make(map[string]string)

// 注册API路由
func register_api(r *mux.Router, path string, f muxhttp.Container) *mux.Route {
	// 注册路由
	apiRoutes[path] = f.Name()
	return r.HandleFunc(path, f)
}
