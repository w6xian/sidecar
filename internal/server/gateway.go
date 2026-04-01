package server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/w6xian/sidecar/internal/protocol"
	"github.com/w6xian/sidecar/pkg/logger"
	"github.com/w6xian/sidecar/pkg/metrics"
)

// GatewayConfig 网关配置
type GatewayConfig struct {
	HTTPAddr     string        `mapstructure:"http_addr"`
	GRPCAddr     string        `mapstructure:"grpc_addr"`
	ProxyTimeout time.Duration `mapstructure:"proxy_timeout"`
}

// Gateway HTTP/gRPC 入口网关
type Gateway struct {
	config  GatewayConfig
	handler *Handler
	router  *Router
	hub     *Hub
}

// NewGateway 创建入口网关
func NewGateway(config GatewayConfig, handler *Handler, router *Router, hub *Hub) *Gateway {
	return &Gateway{
		config:  config,
		handler: handler,
		router:  router,
		hub:     hub,
	}
}

// BuildHTTPRouter 构建 HTTP 路由
func (g *Gateway) BuildHTTPRouter() http.Handler {
	r := chi.NewRouter()

	// 中间件
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(traceIDMiddleware)
	r.Use(middleware.Recoverer)

	// WebSocket 端点
	r.HandleFunc("/sidecar/connect", g.handler.ServeWS)

	// Prometheus 指标端点
	r.Handle("/metrics", promhttp.Handler())

	// 健康检查
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "healthy",
			"time":   time.Now().UTC().Format(time.RFC3339),
		})
	})

	// 路由信息查看（管理用）
	r.Get("/admin/routes", func(w http.ResponseWriter, r *http.Request) {
		routes := g.router.GetAllRoutes()
		result := make([]map[string]interface{}, 0)
		for sid, entry := range routes {
			result = append(result, map[string]interface{}{
				"service_id":     sid,
				"name":           entry.ServiceInfo.Name,
				"protocol":       entry.ServiceInfo.Protocol,
				"expose_path":    entry.ServiceInfo.ExposePath,
				"sidecar_count":  len(entry.ConnIDs),
			})
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})

	// 列出所有在线 Sidecar 连接
	r.Get("/admin/sidecars", g.adminListSidecarsHandler)

	// 管理指令：向指定连接推送更新
	r.Post("/admin/cmd/update/{connID}", g.adminUpdateHandler)

	// 管理指令：向指定连接下发执行 Lua 脚本
	r.Post("/admin/cmd/exec-lua/{connID}", g.adminExecLuaHandler)

	// 管理指令：向指定连接请求上传文件（日志）
	r.Post("/admin/cmd/upload-file/{connID}", g.adminUploadFileHandler)

	// 管理指令：向指定连接写入文件（分段写入）
	r.Post("/admin/cmd/write-file/{connID}", g.adminWriteFileHandler)

	// 管理指令：向指定连接下发执行命令（可执行文件）
	r.Post("/admin/cmd/exec/{connID}", g.adminExecCmdHandler)

	// 连接别名路由 — /{alias}/* 格式
	// 必须放在通配路由之前，因为 chi 的路由匹配是按注册顺序
	r.HandleFunc("/{alias}/*", g.proxyByAliasHandler)

	// 通配代理路由 — 匹配所有注册的服务路径
	r.HandleFunc("/*", g.proxyHTTPHandler)

	return r
}

// proxyByAliasHandler 处理基于连接别名的代理请求 /{alias}/*
func (g *Gateway) proxyByAliasHandler(w http.ResponseWriter, r *http.Request) {
	// 从路径中提取 alias 和子路径
	alias := chi.URLParam(r, "alias")
	path := r.URL.RequestURI()
	traceID := r.Header.Get("X-Trace-Id")
	if traceID == "" {
		traceID = uuid.New().String()
	}

	// 找到该 alias 对应的连接
	sc, found := g.hub.GetConnByAlias(alias)
	if !found {
		http.Error(w, `{"error":"connection not found","alias":"`+alias+`"}`, http.StatusNotFound)
		return
	}

	// 提取子路径：去掉 /{alias} 前缀
	prefix := "/" + alias
	forwardPath := strings.TrimPrefix(path, prefix)
	if forwardPath == "" {
		forwardPath = "/"
	}

	// 读取 body
	var bodyStr string
	if r.Body != nil {
		bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 32*1024*1024))
		if err != nil {
			http.Error(w, `{"error":"read body failed"}`, http.StatusBadRequest)
			return
		}
		bodyStr = base64.StdEncoding.EncodeToString(bodyBytes)
	}

	// 构造 headers
	headers := make(map[string]string)
	for key, values := range r.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}
	headers["X-Trace-Id"] = traceID

	// 构造代理请求
	// 使用该连接注册的第一个服务的 ServiceID（客户端会用它来查找 LocalAddr）
	serviceID := sc.ID
	if len(sc.Services) > 0 {
		// 如果连接注册了服务，使用第一个服务的 ID
		// 客户端会用这个 ID 查找对应的 LocalAddr
		for _, svc := range sc.Services {
			if svc.Protocol == protocol.ProtocolHTTP {
				serviceID = svc.ID
				break
			}
		}
	}

	requestID := uuid.New().String()
	proxyReq := &protocol.ProxyRequest{
		ServiceID: serviceID,
		Protocol:  protocol.ProtocolHTTP,
		HTTP: &protocol.HTTPRequestDetail{
			Method:  r.Method,
			Path:    forwardPath,
			Headers: headers,
			Body:    bodyStr,
		},
	}

	reqMsg, err := protocol.NewMessage(protocol.MsgProxyRequest, requestID, proxyReq)
	if err != nil {
		http.Error(w, `{"error":"encode request failed"}`, http.StatusInternalServerError)
		return
	}

	start := time.Now()
	respMsg, err := sc.SendProxyRequest(reqMsg, g.config.ProxyTimeout)
	if err != nil {
		logger.L().Error("proxy forward failed (by alias)",
			zap.String("alias", alias),
			zap.String("conn_id", sc.ID),
			zap.String("trace_id", traceID),
			zap.Error(err),
		)
		if err == ErrRequestTimeout {
			http.Error(w, `{"error":"gateway timeout"}`, http.StatusGatewayTimeout)
		} else {
			http.Error(w, `{"error":"bad gateway"}`, http.StatusBadGateway)
		}
		return
	}

	duration := time.Since(start).Seconds()
	metrics.ProxyRequestDuration.With(map[string]string{
		"service_id": sc.ID,
		"protocol":   "http",
	}).Observe(duration)

	var proxyResp protocol.ProxyResponse
	if err := json.Unmarshal(respMsg.Payload, &proxyResp); err != nil {
		logger.L().Error("decode proxy response failed",
			zap.String("alias", alias),
			zap.String("trace_id", traceID),
			zap.Error(err),
		)
		http.Error(w, `{"error":"decode response failed"}`, http.StatusBadGateway)
		return
	}

	if proxyResp.HTTP == nil {
		http.Error(w, `{"error":"invalid response from sidecar"}`, http.StatusBadGateway)
		return
	}

	statusStr := "success"
	if proxyResp.HTTP.StatusCode >= 400 {
		statusStr = "error"
	}
	metrics.ProxyRequestsTotal.With(map[string]string{
		"service_id": sc.ID,
		"protocol":   "http",
		"method":     r.Method,
		"status":     statusStr,
	}).Inc()

	// 回写响应头
	for k, v := range proxyResp.HTTP.Headers {
		w.Header().Set(k, v)
	}
	w.Header().Set("X-Trace-Id", traceID)

	// 回写状态码
	w.WriteHeader(proxyResp.HTTP.StatusCode)

	// 回写 body
	if proxyResp.HTTP.Body != "" {
		bodyData, err := base64.StdEncoding.DecodeString(proxyResp.HTTP.Body)
		if err != nil {
			logger.L().Error("decode response body failed",
				zap.String("alias", alias),
				zap.String("trace_id", traceID),
				zap.Error(err),
			)
			return
		}
		w.Write(bodyData)
	}
}

// proxyHTTPHandler 处理 HTTP 代理请求
func (g *Gateway) proxyHTTPHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.RequestURI()
	traceID := r.Header.Get("X-Trace-Id")
	if traceID == "" {
		traceID = uuid.New().String()
	}

	// 路径匹配
	entry, matchedPrefix, found := g.router.MatchByPath(path)
	if !found {
		http.Error(w, `{"error":"service not found","path":"`+path+`"}`, http.StatusNotFound)
		return
	}

	// 构造转发的路径：去掉服务暴露前缀
	forwardPath := strings.TrimPrefix(path, matchedPrefix)
	if forwardPath == "" {
		forwardPath = "/"
	}

	// 读取 body
	var bodyStr string
	if r.Body != nil {
		bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 32*1024*1024))
		if err != nil {
			http.Error(w, `{"error":"read body failed"}`, http.StatusBadRequest)
			return
		}
		bodyStr = base64.StdEncoding.EncodeToString(bodyBytes)
	}

	// 构造 headers
	headers := make(map[string]string)
	for key, values := range r.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}
	headers["X-Trace-Id"] = traceID

	// 构造代理请求
	proxyReq := &protocol.ProxyRequest{
		ServiceID: entry.ServiceInfo.ID,
		Protocol:  protocol.ProtocolHTTP,
		HTTP: &protocol.HTTPRequestDetail{
			Method:  r.Method,
			Path:    forwardPath,
			Headers: headers,
			Body:    bodyStr,
		},
	}

	resp, err := g.handler.ForwardHTTP(entry.ServiceInfo.ID, proxyReq)
	if err != nil {
		logger.L().Error("proxy forward failed",
			zap.String("service_id", entry.ServiceInfo.ID),
			zap.String("trace_id", traceID),
			zap.Error(err),
		)
		if err == ErrRequestTimeout {
			http.Error(w, `{"error":"gateway timeout"}`, http.StatusGatewayTimeout)
		} else {
			http.Error(w, `{"error":"bad gateway"}`, http.StatusBadGateway)
		}
		return
	}

	if resp.HTTP == nil {
		http.Error(w, `{"error":"invalid response from sidecar"}`, http.StatusBadGateway)
		return
	}

	// 回写响应头
	for k, v := range resp.HTTP.Headers {
		w.Header().Set(k, v)
	}
	w.Header().Set("X-Trace-Id", traceID)

	// 回写状态码
	w.WriteHeader(resp.HTTP.StatusCode)

	// 回写 body
	if resp.HTTP.Body != "" {
		bodyData, err := base64.StdEncoding.DecodeString(resp.HTTP.Body)
		if err != nil {
			logger.L().Error("decode response body failed", zap.Error(err))
			return
		}
		w.Write(bodyData)
	}
}

// StartHTTP 启动 HTTP 网关
func (g *Gateway) StartHTTP() error {
	httpRouter := g.BuildHTTPRouter()
	srv := &http.Server{
		Addr:         g.config.HTTPAddr,
		Handler:      httpRouter,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	logger.L().Info("HTTP gateway starting", zap.String("addr", g.config.HTTPAddr))
	return srv.ListenAndServe()
}

// StartGRPC 启动 gRPC 入口网关
func (g *Gateway) StartGRPC() error {
	lis, err := net.Listen("tcp", g.config.GRPCAddr)
	if err != nil {
		return fmt.Errorf("grpc listen: %w", err)
	}

	srv := grpc.NewServer(
		grpc.UnknownServiceHandler(g.grpcProxyHandler),
	)

	logger.L().Info("gRPC gateway starting", zap.String("addr", g.config.GRPCAddr))
	return srv.Serve(lis)
}

// grpcProxyHandler 通用 gRPC 代理处理器
func (g *Gateway) grpcProxyHandler(srv interface{}, stream grpc.ServerStream) error {
	fullMethod, ok := grpc.MethodFromServerStream(stream)
	if !ok {
		return status.Error(codes.Internal, "failed to get method name")
	}

	// 根据 gRPC 方法名匹配路由
	// gRPC 方法格式: /package.Service/Method
	// 我们使用注册的 expose_path 作为前缀匹配
	entry, _, found := g.router.MatchByPath(fullMethod)
	if !found {
		return status.Errorf(codes.Unimplemented, "unknown service: %s", fullMethod)
	}

	// 接收请求数据
	var reqData []byte
	msg := &rawFrame{}
	if err := stream.RecvMsg(msg); err != nil {
		return status.Errorf(codes.Internal, "recv request: %v", err)
	}
	reqData = msg.data

	// 选择 Sidecar 连接
	connID := g.router.PickConn(entry)
	if connID == "" {
		return status.Error(codes.Unavailable, "no available sidecar")
	}

	sc, ok := g.hub.GetConn(connID)
	if !ok {
		return status.Error(codes.Unavailable, "sidecar connection lost")
	}

	// 构造代理请求
	proxyReq := &protocol.ProxyRequest{
		ServiceID: entry.ServiceInfo.ID,
		Protocol:  protocol.ProtocolGRPC,
		GRPC: &protocol.GRPCRequestDetail{
			FullMethod: fullMethod,
			Data:       base64.StdEncoding.EncodeToString(reqData),
		},
	}

	requestID := uuid.New().String()
	reqMsg, err := protocol.NewMessage(protocol.MsgProxyRequest, requestID, proxyReq)
	if err != nil {
		return status.Errorf(codes.Internal, "encode request: %v", err)
	}

	start := time.Now()
	respMsg, err := sc.SendProxyRequest(reqMsg, g.config.ProxyTimeout)
	if err != nil {
		metrics.ProxyRequestsTotal.With(map[string]string{
			"service_id": entry.ServiceInfo.ID,
			"protocol":   "grpc",
			"method":     fullMethod,
			"status":     "error",
		}).Inc()
		if err == ErrRequestTimeout {
			return status.Error(codes.DeadlineExceeded, "proxy timeout")
		}
		return status.Errorf(codes.Internal, "proxy error: %v", err)
	}

	duration := time.Since(start).Seconds()
	metrics.ProxyRequestDuration.With(map[string]string{
		"service_id": entry.ServiceInfo.ID,
		"protocol":   "grpc",
	}).Observe(duration)

	var proxyResp protocol.ProxyResponse
	if err := json.Unmarshal(respMsg.Payload, &proxyResp); err != nil {
		return status.Errorf(codes.Internal, "decode response: %v", err)
	}

	if proxyResp.GRPC == nil {
		return status.Error(codes.Internal, "invalid grpc response from sidecar")
	}

	if proxyResp.GRPC.StatusCode != 0 {
		metrics.ProxyRequestsTotal.With(map[string]string{
			"service_id": entry.ServiceInfo.ID,
			"protocol":   "grpc",
			"method":     fullMethod,
			"status":     "error",
		}).Inc()
		return status.Error(codes.Code(proxyResp.GRPC.StatusCode), proxyResp.GRPC.Message)
	}

	respData, err := base64.StdEncoding.DecodeString(proxyResp.GRPC.Data)
	if err != nil {
		return status.Errorf(codes.Internal, "decode response data: %v", err)
	}

	metrics.ProxyRequestsTotal.With(map[string]string{
		"service_id": entry.ServiceInfo.ID,
		"protocol":   "grpc",
		"method":     fullMethod,
		"status":     "success",
	}).Inc()

	return stream.SendMsg(&rawFrame{data: respData})
}

// rawFrame 用于传递 gRPC 原始帧
type rawFrame struct {
	data []byte
}

func (f *rawFrame) Reset()         {}
func (f *rawFrame) String() string { return string(f.data) }
func (f *rawFrame) ProtoMessage()  {}
func (f *rawFrame) Marshal() ([]byte, error) {
	return f.data, nil
}
func (f *rawFrame) Unmarshal(b []byte) error {
	f.data = b
	return nil
}

// traceIDMiddleware 注入 TraceID
func traceIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := r.Header.Get("X-Trace-Id")
		if traceID == "" {
			traceID = uuid.New().String()
			r.Header.Set("X-Trace-Id", traceID)
		}
		w.Header().Set("X-Trace-Id", traceID)
		next.ServeHTTP(w, r)
	})
}

// adminUpdateHandler POST /admin/cmd/update/{connID}
// Body JSON: protocol.UpdatePayload
func (g *Gateway) adminUpdateHandler(w http.ResponseWriter, r *http.Request) {
	connID := chi.URLParam(r, "connID")
	var payload protocol.UpdatePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	ack, err := g.handler.SendUpdate(connID, &payload)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(ack)
}

// adminExecLuaHandler POST /admin/cmd/exec-lua/{connID}
// Body JSON: protocol.ExecLuaPayload
func (g *Gateway) adminExecLuaHandler(w http.ResponseWriter, r *http.Request) {
	connID := chi.URLParam(r, "connID")
	var payload protocol.ExecLuaPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	result, err := g.handler.SendExecLua(connID, &payload)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(result)
}

// adminListSidecarsHandler GET /admin/sidecars
// 返回所有在线 Sidecar 连接列表（connID、alias、服务清单）
func (g *Gateway) adminListSidecarsHandler(w http.ResponseWriter, r *http.Request) {
	list := g.hub.ListConns()
	if list == nil {
		list = []SidecarInfo{} // 确保返回 [] 而非 null
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"total":   len(list),
		"sidecars": list,
	})
}

// adminUploadFileHandler POST /admin/cmd/upload-file/{connID}
// Body JSON: protocol.UploadFilePayload
func (g *Gateway) adminUploadFileHandler(w http.ResponseWriter, r *http.Request) {
	connID := chi.URLParam(r, "connID")
	var payload protocol.UploadFilePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	result, err := g.handler.SendUploadFile(connID, &payload)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(result)
}

// adminWriteFileHandler POST /admin/cmd/write-file/{connID}
// 向 Sidecar 写入文件，支持分块传输。
// Body JSON: WriteFileRequest
//
//	{
//	  "path": "/etc/config/app.conf",    // 目标路径（客户端本地）
//	  "data": "<base64 encoded content>", // 文件完整内容（base64）
//	  "perm": 420,                        // 文件权限（可选，默认 0644=420）
//	  "chunk_size": 262144                // 每块字节数（可选，默认 256KB）
//	}
func (g *Gateway) adminWriteFileHandler(w http.ResponseWriter, r *http.Request) {
	connID := chi.URLParam(r, "connID")
	var req WriteFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if req.Path == "" {
		http.Error(w, `{"error":"path is required"}`, http.StatusBadRequest)
		return
	}
	if req.Data == "" {
		http.Error(w, `{"error":"data is required"}`, http.StatusBadRequest)
		return
	}

	result, err := g.handler.SendWriteFile(connID, &req)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	if !result.Success {
		w.WriteHeader(http.StatusInternalServerError)
	}
	json.NewEncoder(w).Encode(result)
}

// adminExecCmdHandler POST /admin/cmd/exec/{connID}
// 向指定 Sidecar 下发执行命令（可执行文件）指令。
// Body JSON: protocol.ExecCmdPayload
//
//	{
//	  "command": "/usr/local/bin/myapp",  // 可执行文件路径或命令
//	  "args": ["--flag", "value"],        // 参数列表
//	  "env": {"KEY": "VALUE"},            // 环境变量（可选）
//	  "dir": "/tmp",                      // 工作目录（可选）
//	  "timeout": 30,                      // 超时秒数（可选，默认 60s）
//	  "stdin": "input data"               // 标准输入（可选）
//	}
func (g *Gateway) adminExecCmdHandler(w http.ResponseWriter, r *http.Request) {
	connID := chi.URLParam(r, "connID")
	var payload protocol.ExecCmdPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if payload.Command == "" {
		http.Error(w, `{"error":"command is required"}`, http.StatusBadRequest)
		return
	}

	result, err := g.handler.SendExecCmd(connID, &payload)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(result)
}
