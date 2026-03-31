package sidecar

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/w6xian/sidecar/internal/config"
	"github.com/w6xian/sidecar/internal/protocol"
	"github.com/w6xian/sidecar/pkg/logger"
)

// Forwarder 本地请求转发器
type Forwarder struct {
	services   map[string]config.ServiceConfig
	httpClient *http.Client
}

// NewForwarder 创建转发器
func NewForwarder(services []config.ServiceConfig) *Forwarder {
	svcMap := make(map[string]config.ServiceConfig, len(services))
	for _, svc := range services {
		svcMap[svc.Id] = svc
	}

	return &Forwarder{
		services: svcMap,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

// Forward 根据协议类型转发请求
func (f *Forwarder) Forward(reqPayload *protocol.ProxyRequest) (*protocol.ProxyResponse, error) {
	svc, exists := f.services[reqPayload.ServiceID]
	if !exists {
		return nil, fmt.Errorf("unknown service: %s", reqPayload.ServiceID)
	}

	switch reqPayload.Protocol {
	case protocol.ProtocolHTTP:
		return f.forwardHTTP(svc, reqPayload.HTTP)
	case protocol.ProtocolGRPC:
		return f.forwardGRPC(svc, reqPayload.GRPC)
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", reqPayload.Protocol)
	}
}

// forwardHTTP 转发 HTTP 请求到本地服务
func (f *Forwarder) forwardHTTP(svc config.ServiceConfig, detail *protocol.HTTPRequestDetail) (*protocol.ProxyResponse, error) {
	if detail == nil {
		return nil, fmt.Errorf("http detail is nil")
	}

	// 构造本地请求 URL
	url := svc.LocalAddr + detail.Path

	// 解码 body
	var bodyReader io.Reader
	if detail.Body != "" {
		bodyBytes, err := base64.StdEncoding.DecodeString(detail.Body)
		if err != nil {
			return nil, fmt.Errorf("decode request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// 创建请求
	req, err := http.NewRequest(detail.Method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// 设置 headers
	for k, v := range detail.Headers {
		req.Header.Set(k, v)
	}

	// 发送请求
	logger.L().Debug("forwarding HTTP request",
		zap.String("service_id", svc.Id),
		zap.String("method", detail.Method),
		zap.String("url", url),
	)

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应 body
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 32*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	// 构造响应 headers
	respHeaders := make(map[string]string)
	for k, v := range resp.Header {
		if len(v) > 0 {
			respHeaders[k] = v[0]
		}
	}

	return &protocol.ProxyResponse{
		Protocol: protocol.ProtocolHTTP,
		HTTP: &protocol.HTTPResponseDetail{
			StatusCode: resp.StatusCode,
			Headers:    respHeaders,
			Body:       base64.StdEncoding.EncodeToString(respBody),
		},
	}, nil
}

// forwardGRPC 转发 gRPC 请求到本地服务
func (f *Forwarder) forwardGRPC(svc config.ServiceConfig, detail *protocol.GRPCRequestDetail) (*protocol.ProxyResponse, error) {
	if detail == nil {
		return nil, fmt.Errorf("grpc detail is nil")
	}

	// 建立 gRPC 连接
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, svc.LocalAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("grpc dial: %w", err)
	}
	defer conn.Close()

	// 解码请求数据
	reqData, err := base64.StdEncoding.DecodeString(detail.Data)
	if err != nil {
		return nil, fmt.Errorf("decode grpc request data: %w", err)
	}

	// Unary RPC 调用
	respFrame := &rawFrame{}
	err = conn.Invoke(ctx, detail.FullMethod, &rawFrame{data: reqData}, respFrame)
	if err != nil {
		st, ok := statusFromError(err)
		if ok {
			return &protocol.ProxyResponse{
				Protocol: protocol.ProtocolGRPC,
				GRPC: &protocol.GRPCResponseDetail{
					StatusCode: int(st.Code()),
					Message:    st.Message(),
				},
			}, nil
		}
		return nil, fmt.Errorf("grpc invoke: %w", err)
	}

	return &protocol.ProxyResponse{
		Protocol: protocol.ProtocolGRPC,
		GRPC: &protocol.GRPCResponseDetail{
			StatusCode: 0, // OK
			Data:       base64.StdEncoding.EncodeToString(respFrame.data),
		},
	}, nil
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

// statusFromError 从 gRPC 错误中提取 status
func statusFromError(err error) (interface {
	Code() uint32
	Message() string
}, bool) {
	type grpcStatus struct {
		code    uint32
		message string
	}

	// 尝试使用 json 编解码来获取错误信息
	data, jsonErr := json.Marshal(err)
	if jsonErr == nil {
		var result map[string]interface{}
		if json.Unmarshal(data, &result) == nil {
			if code, ok := result["code"]; ok {
				return &simpleStatus{
					code:    uint32(code.(float64)),
					message: fmt.Sprintf("%v", result["message"]),
				}, true
			}
		}
	}

	return nil, false
}

type simpleStatus struct {
	code    uint32
	message string
}

func (s *simpleStatus) Code() uint32    { return s.code }
func (s *simpleStatus) Message() string { return s.message }
