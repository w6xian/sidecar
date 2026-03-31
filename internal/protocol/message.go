package protocol

import "encoding/json"

// Message 是所有 WebSocket 消息的通用信封
type Message struct {
	Type      MessageType     `json:"type"`
	RequestID string          `json:"request_id,omitempty"`
	Token     string          `json:"token,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Error     string          `json:"error,omitempty"`
	AppId     string          `json:"app_id,omitempty"`
	Sign      string          `json:"sign,omitempty"`
	Timestamp int64           `json:"timestamp,omitempty"`
}

// ServiceInfo 描述一个注册的服务
type ServiceInfo struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Protocol   Protocol `json:"protocol"`
	ExposePath string   `json:"expose_path"`
}

// RegisterRequest 注册请求载荷
type RegisterRequest struct {
	Services  []ServiceInfo `json:"services"`
	ConnAlias string        `json:"conn_alias,omitempty"` // 连接别名，用于 /{alias}/* 路由
}

// RegisterACK 注册确认载荷
type RegisterACK struct {
	Success   bool     `json:"success"`
	Message   string   `json:"message,omitempty"`
	Accepted  []string `json:"accepted,omitempty"`   // 成功注册的 serviceId 列表
	ConnAlias string   `json:"conn_alias,omitempty"` // 服务端确认的连接别名（可能与请求不同）
}

// HTTPRequestDetail HTTP 请求详情
type HTTPRequestDetail struct {
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"` // base64 编码的 body
}

// HTTPResponseDetail HTTP 响应详情
type HTTPResponseDetail struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       string            `json:"body,omitempty"` // base64 编码的 body
}

// GRPCRequestDetail gRPC 请求详情
type GRPCRequestDetail struct {
	FullMethod string `json:"full_method"` // e.g. /package.Service/Method
	Data       string `json:"data"`        // base64 编码的 protobuf 帧
}

// GRPCResponseDetail gRPC 响应详情
type GRPCResponseDetail struct {
	Data       string `json:"data"`        // base64 编码的 protobuf 帧
	StatusCode int    `json:"status_code"` // gRPC status code
	Message    string `json:"message,omitempty"`
}

// ProxyRequest 代理请求
type ProxyRequest struct {
	ServiceID string             `json:"service_id"`
	Protocol  Protocol           `json:"protocol"`
	HTTP      *HTTPRequestDetail `json:"http,omitempty"`
	GRPC      *GRPCRequestDetail `json:"grpc,omitempty"`
}

// ProxyResponse 代理响应
type ProxyResponse struct {
	Protocol Protocol            `json:"protocol"`
	HTTP     *HTTPResponseDetail `json:"http,omitempty"`
	GRPC     *GRPCResponseDetail `json:"grpc,omitempty"`
}

// HeartbeatPayload 心跳载荷
type HeartbeatPayload struct {
	Timestamp int64 `json:"timestamp"`
}

// UnregisterRequest 注销请求
type UnregisterRequest struct {
	ServiceIDs []string `json:"service_ids"`
}

// ErrorPayload 错误信息
type ErrorPayload struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// EncodePayload 将载荷序列化为 json.RawMessage
func EncodePayload(v interface{}) (json.RawMessage, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}

// NewMessage 创建一条新消息
func NewMessage(msgType MessageType, requestID string, payload interface{}) (*Message, error) {
	msg := &Message{
		Type:      msgType,
		RequestID: requestID,
	}
	if payload != nil {
		p, err := EncodePayload(payload)
		if err != nil {
			return nil, err
		}
		msg.Payload = p
	}
	return msg, nil
}

// NewErrorMessage 创建一条错误消息
func NewErrorMessage(requestID string, code int, message string) (*Message, error) {
	return NewMessage(MsgError, requestID, &ErrorPayload{
		Code:    code,
		Message: message,
	})
}
