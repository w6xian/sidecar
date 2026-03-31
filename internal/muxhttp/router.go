package muxhttp

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/w6xian/sidecar/internal/utils"
	"github.com/w6xian/sidecar/internal/utils/id"

	"github.com/julienschmidt/httprouter"
)

type RouterHandler func(http.ResponseWriter, *http.Request, ...httprouter.Param) ([]byte, error)
type RouterDecorator func(RouterHandler) RouterHandler

func RouterDecorate(f RouterHandler, ds ...RouterDecorator) httprouter.Handle {
	decorated := f
	ds = routerReverse(ds)
	// <- 从右向左
	ds = append(ds, RouterRequestId)
	// 嵌套
	for _, decorate := range ds {
		// 本质上handler,在最右边一个里扫行，可以实现前置与后置
		decorated = decorate(decorated)
	}
	return func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
		decorated(w, req, ps...)
	}
}

func routerReverse[T RouterDecorator](slice []T) []T {
	for i, j := 0, len(slice)-1; i < j; i, j = i+1, j-1 {
		slice[i], slice[j] = slice[j], slice[i]
	}
	return slice
}
func RouterV2Response(f RouterHandler) RouterHandler {
	return func(w http.ResponseWriter, req *http.Request, ps ...httprouter.Param) ([]byte, error) {
		fmt.Println("v2")
		r, e := f(w, req)
		fmt.Println(r, e)
		if e != nil {
			ne := NewErr(e)
			er, _ := ne.ToBytes()
			w.WriteHeader(http.StatusOK)
			w.Write(er)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write(r)
		}
		return []byte{}, nil
	}
}

func RouterRequestId(f RouterHandler) RouterHandler {
	return func(w http.ResponseWriter, r *http.Request, ps ...httprouter.Param) ([]byte, error) {
		reqId, err := id.NextId(1)
		if err != nil {
			reqId = time.Now().UnixNano()
		}
		ctx := context.WithValue(r.Context(), ContextKey("request_id"), reqId)
		r = r.WithContext(ctx)
		return f(w, r)
	}
}

func RouterCros(f RouterHandler) RouterHandler {
	return func(w http.ResponseWriter, r *http.Request, ps ...httprouter.Param) ([]byte, error) {
		header := w.Header()
		rHeader := r.Header
		header.Set("Access-Control-Allow-Methods", "POST,GET,PUT,DELETE,PATCH,OPTIONS")
		header.Set("Access-Control-Allow-Origin", rHeader.Get("Origin"))
		header.Set("Access-Control-Allow-Credentials", "true")
		header.Set("Access-Control-Allow-Headers", "Origin, Cookie, X-File-Name,X-Requested-With, Content-Type, Accept,Authorization,Orz-Auth3,Orz-Auth4,Orz-Ref3,Orz-Ref4,Orz-API-Version,Orz-Name,Orz-Agents,Orz-Agents-Code,Orz-Agents-Service,Orz-Agents-Name,Orz-Agents-App,Orz-Extra-Columns")
		header.Set("Access-Control-Expose-Headers", "ORZ-PROXY-money_decimal,ORZ-PROXY-money_symbol,ORZ-PROXY-money_format,ORZ-PROXY-use_subtotal_symbol,ORZ-PROXY-print_money_symbol,ORZ-PROXY-__shop_id,ORZ-PROXY-__channel,Orz-Extra-Columns")
		header.Set("P3P", "CP=CAO PSA OUR")
		return f(w, r)
	}
}

func RouterAuthOrg3(f RouterHandler) RouterHandler {
	return func(w http.ResponseWriter, r *http.Request, ps ...httprouter.Param) ([]byte, error) {
		// Orz-Auth3: orz:L/394733145922399002G/2130766619;no-cache
		token := r.Header.Get("ORZ-AUTH3")
		if token != "" {
			session := utils.Orz3Decode(token)
			ctx := context.WithValue(r.Context(), ContextKey("orz3"), session)
			r = r.WithContext(ctx)
		}
		return f(w, r)
	}
}

func RouterAuthCookie(f RouterHandler) RouterHandler {
	return func(w http.ResponseWriter, req *http.Request, ps ...httprouter.Param) ([]byte, error) {
		// Orz-Auth3: orz:L/394733145922399002G/2130766619;no-cache
		token, err := req.Cookie("orzj-session")
		if err == nil {
			ctx := context.WithValue(req.Context(), ContextKey("orz"), token.Value)
			req = req.WithContext(ctx)
		}
		return f(w, req, ps...)
	}
}

func RouterJsonValue(f RouterHandler) RouterHandler {
	return func(w http.ResponseWriter, req *http.Request, ps ...httprouter.Param) ([]byte, error) {
		defer func() {
			if err := recover(); err != nil {
				fmt.Println("recover", err)
			}
		}()
		// Orz-Auth3: orz:L/394733145922399002G/2130766619;no-cache
		js, _ := io.ReadAll(req.Body)
		hpj := &HttpPostJson{
			Data: js,
		}
		ctx := context.WithValue(req.Context(), ContextKey("post_value"), hpj)
		req = req.WithContext(ctx)
		return f(w, req, ps...)
	}
}
