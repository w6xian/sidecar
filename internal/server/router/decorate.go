package router

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/w6xian/sidecar/internal/lager"
	"github.com/w6xian/sidecar/internal/muxhttp"
	"github.com/w6xian/sidecar/internal/utils/id"
)

type contextKey string

func (k contextKey) String() string { return "net/api context value " + string(k) }

var (
	// ContextKeyOrz3 is the context key for orz3 session.
	ContextKeyOrz3    = contextKey("orz3")
	ContextKeyCancels = contextKey("cancels")
	ContextKeyClaims  = contextKey("claims")
	ContextKeyLog     = contextKey("log")
)

func ExcelV2(h muxhttp.Handler, ds ...muxhttp.Decorator) muxhttp.Container {
	// 第一阶段的decorator
	l1 := []muxhttp.Decorator{RequestId, I18nMiddleware, muxhttp.JsonValue}
	// 第二阶段的decorator
	l2 := []muxhttp.Decorator{}
	// 合并自定义的decorator
	l2 = append(l2, ds...)
	// 第三阶段的decorator
	l3 := []muxhttp.Decorator{lager.RequestLager}

	pre := append(l1, l2...)
	//优先执行自定义的pre
	pre = append(pre, l3...)
	pre = append(pre, muxhttp.ExcelResponse)
	return muxhttp.Decorate(h, pre...)
}

func StaticsV2(h muxhttp.Handler, ds ...muxhttp.Decorator) muxhttp.Container {
	// 第一阶段的decorator
	l1 := []muxhttp.Decorator{I18nMiddleware}
	// 第二阶段的decorator
	l2 := []muxhttp.Decorator{}
	// 合并自定义的decorator
	l2 = append(l2, ds...)
	// 第三阶段的decorator
	l3 := []muxhttp.Decorator{lager.RequestLager}

	pre := append(l1, l2...)
	//优先执行自定义的pre
	pre = append(pre, l3...)
	return muxhttp.Decorate(h, pre...)
}

func JsonV2(h muxhttp.Handler, ds ...muxhttp.Decorator) muxhttp.Container {
	// 第一阶段的decorator
	l1 := []muxhttp.Decorator{RequestId, I18nMiddleware, muxhttp.JsonValue}
	// 第二阶段的decorator
	l2 := []muxhttp.Decorator{}
	// 合并自定义的decorator
	l2 = append(l2, ds...)
	// 第三阶段的decorator
	l3 := []muxhttp.Decorator{lager.RequestLager}

	pre := append(l1, l2...)
	//优先执行自定义的pre
	pre = append(pre, l3...)
	pre = append(pre, muxhttp.V2Response)
	return muxhttp.Decorate(h, pre...)
}

func FileV2(h muxhttp.Handler, ds ...muxhttp.Decorator) muxhttp.Container {
	// 第一阶段的decorator
	l1 := []muxhttp.Decorator{RequestId, I18nMiddleware}
	// 第二阶段的decorator
	l2 := []muxhttp.Decorator{}
	// 合并自定义的decorator
	l2 = append(l2, ds...)
	// 第三阶段的decorator
	l3 := []muxhttp.Decorator{lager.RequestLager}

	pre := append(l1, l2...)
	//优先执行自定义的pre
	pre = append(pre, l3...)
	pre = append(pre, muxhttp.V2Response)
	return muxhttp.Decorate(h, pre...)
}

var nodeId = int64(1)
var locker = &sync.Mutex{}

// Http Request 加Id
func RequestId(f muxhttp.Handler) muxhttp.Handler {
	return func(w http.ResponseWriter, r *http.Request) ([]byte, error) {
		locker.Lock()
		defer locker.Unlock()
		reqId, err := id.NextId(nodeId)
		if err != nil {
			reqId = time.Now().UnixNano()
		}
		ctx := context.WithValue(r.Context(), GetRequestIdKey(), reqId)
		r = r.WithContext(ctx)
		return f(w, r)
	}
}

func I18nMiddleware(f muxhttp.Handler) muxhttp.Handler {
	return func(w http.ResponseWriter, r *http.Request) ([]byte, error) {
		lang := detectLanguage(r)
		ctx := r.Context()
		// 将语言信息保存到上下文中
		ctx = context.WithValue(ctx, GetLanguageKey(), lang)
		r = r.WithContext(ctx)
		return f(w, r)
	}
}

// 语言检测策略：
// 1. 首先检查查询参数 ?lang=xx
// 2. 然后检查 Cookie
// 3. 最后检查 Accept-Language 头
func detectLanguage(r *http.Request) string {
	// 1. 检查查询参数
	queryLang := r.URL.Query().Get("lang")
	if queryLang != "" {
		return queryLang
	}

	// 2. 检查 Cookie
	langCookie, err := r.Cookie("language")
	if err == nil && langCookie.Value != "" {
		return langCookie.Value
	}

	// 3. 检查 Accept-Language 头
	acceptLang := r.Header.Get("Accept-Language")
	if acceptLang != "" {
		// 提取首选语言
		langs := strings.Split(acceptLang, ",")
		if len(langs) > 0 {
			// 提取语言代码 (en-US -> en)
			return strings.Split(langs[0], "-")[0]
		}
	}

	// 默认返回英语
	return "en"
}
