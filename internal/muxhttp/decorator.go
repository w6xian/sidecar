package muxhttp

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/w6xian/sidecar/internal/utils"
	"github.com/w6xian/sidecar/internal/utils/id"
)

type Handler func(http.ResponseWriter, *http.Request) ([]byte, error)

type Container func(http.ResponseWriter, *http.Request)
type Decorator func(Handler) Handler
type ContextKey string

func (k ContextKey) String() string { return "net/http context value " + string(k) }

func (c Container) Name() string {
	return fmt.Sprintf("%T", c)
}

func Decorate(f Handler, ds ...Decorator) Container {
	decorated := f
	ds = reverse(ds)
	// <- 从右向左
	ds = append(ds, RequestId)
	// 嵌套
	for _, decorate := range ds {
		// 本质上handler,在最右边一个里扫行，可以实现前置与后置
		decorated = decorate(decorated)
	}
	return func(w http.ResponseWriter, req *http.Request) {
		decorated(w, req)
	}
}

func reverse[T Decorator](slice []T) []T {
	for i, j := 0, len(slice)-1; i < j; i, j = i+1, j-1 {
		slice[i], slice[j] = slice[j], slice[i]
	}
	return slice
}

func Decorater(f Handler, ds ...Decorator) Container {
	decorated := f
	// 嵌套
	for _, decorate := range ds {
		// 本质上handler,在最右边一个里扫行，可以实现前置与后置
		decorated = decorate(decorated)
	}
	return func(w http.ResponseWriter, req *http.Request) {
		decorated(w, req)
	}
}

type Empty []byte

func HttpCors(f Handler) Handler {
	return func(w http.ResponseWriter, r *http.Request) ([]byte, error) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return nil, nil
		} else {
			return f(w, r)
		}
	}
}

// // io.ReadAll 是一次性读取，也只能读一次，应当注意，不要重复读取
func JsonValue(f Handler) Handler {
	return func(w http.ResponseWriter, r *http.Request) ([]byte, error) {
		// Orz-Auth3: orz:L/394733145922399002G/2130766619;no-cache
		defer func() {
			if err := recover(); err != nil {
				fmt.Println("recover", err)
			}
		}()
		js, _ := io.ReadAll(r.Body)
		// fmt.Println("io.ReadALL", string(js))
		hpj := &HttpPostJson{
			Data: js,
		}
		// fmt.Println("JsonValue", hpj)
		ctx := context.WithValue(r.Context(), ContextKey("post_value"), hpj)
		r = r.WithContext(ctx)
		rst, err := f(w, r)
		// fmt.Println("JsonValue 2")
		return rst, err
	}
}

func RequestId(f Handler) Handler {
	return func(w http.ResponseWriter, r *http.Request) ([]byte, error) {
		reqId, err := id.NextId(1)
		if err != nil {
			reqId = time.Now().UnixNano()
		}
		ctx := context.WithValue(r.Context(), ContextKey("request_id"), reqId)
		r = r.WithContext(ctx)
		return f(w, r)
	}
}

func AuthOrg3(f Handler) Handler {
	return func(w http.ResponseWriter, r *http.Request) ([]byte, error) {
		// Orz-Auth3: orz:L/394733145922399002G/2130766619;no-cache
		token := r.Header.Get("ORZ-AUTH3")
		if token != "" {
			session := utils.Orz3Decode(token)
			ctx := context.WithValue(r.Context(), ContextKey("orz3"), session)
			r = r.WithContext(ctx)
		}
		// fmt.Println("AuthOrg3")
		return f(w, r)
	}
}

func AuthCookie(f Handler) Handler {
	return func(w http.ResponseWriter, req *http.Request) ([]byte, error) {
		// Orz-Auth3: orz:L/394733145922399002G/2130766619;no-cache
		token, err := req.Cookie("orzj-session")
		// fmt.Println("AuthCookie", "token")
		if err == nil {
			ctx := context.WithValue(req.Context(), ContextKey("orz"), token.Value)
			req = req.WithContext(ctx)
		}
		return f(w, req)
	}
}
