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

// ----------- 更新指令 -----------

// UpdatePayload 更新指令载荷（Server → Local）
type UpdatePayload struct {
	Version  string `json:"version"`            // 目标版本号，例如 "1.2.3"
	DownURL  string `json:"down_url"`            // 新版本下载地址
	Checksum string `json:"checksum,omitempty"` // 文件 SHA256 校验值（可选）
	Force    bool   `json:"force,omitempty"`    // 是否强制更新（不等待空闲）
}

// UpdateACKPayload 更新响应载荷（Local → Server）
type UpdateACKPayload struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Version string `json:"version,omitempty"` // 更新后的实际版本
}

// ----------- 执行 Lua 脚本 -----------

// ExecLuaPayload 执行 Lua 脚本载荷（Server → Local）
type ExecLuaPayload struct {
	Script  string            `json:"script"`            // Lua 脚本内容（与 file 二选一）
	File    string            `json:"file,omitempty"`    // 本地脚本文件路径（与 script 二选一）
	Timeout int               `json:"timeout,omitempty"` // 执行超时秒数，0 表示默认 30s
	Env     map[string]string `json:"env,omitempty"`     // 传入 Lua 的全局变量
}

// ExecLuaResultPayload Lua 脚本执行结果（Local → Server）
type ExecLuaResultPayload struct {
	Success bool   `json:"success"`
	Output  string `json:"output,omitempty"`  // 标准输出捕获
	Error   string `json:"error,omitempty"`   // 错误信息
	Elapsed int64  `json:"elapsed,omitempty"` // 执行耗时（毫秒）
}

// ----------- 上传文件（日志） -----------

// UploadFilePayload 上传文件请求（Server → Local）
type UploadFilePayload struct {
	Path     string `json:"path"`               // 要上传的本地文件路径
	MaxBytes int64  `json:"max_bytes,omitempty"` // 最多读取字节数，0 表示全部（最大 32MB）
	Tail     int    `json:"tail,omitempty"`      // 读取末尾行数（>0 时仅读尾部），0 表示读全部
}

// UploadFileDataPayload 文件数据（Local → Server）
type UploadFileDataPayload struct {
	Success  bool   `json:"success"`
	Path     string `json:"path"`              // 原始路径（回显）
	Data     string `json:"data,omitempty"`    // base64 编码文件内容
	Size     int64  `json:"size,omitempty"`    // 实际读取字节数
	Error    string `json:"error,omitempty"`   // 错误信息
	Truncated bool  `json:"truncated,omitempty"` // 是否因 MaxBytes 被截断
}

// ----------- 写文件（分段写入） -----------

// WriteFileChunkPayload 写文件分块载荷（Server → Local）
// 服务端按序发送多个分块，客户端按 Seq 顺序写入文件。
// Seq==0 时创建/截断文件（并自动创建目录），Seq>0 时追加写入。
// 最后一块 IsFinal==true，客户端写完后关闭文件并回 ACK。
type WriteFileChunkPayload struct {
	Path    string `json:"path"`              // 目标文件绝对路径（客户端本地）
	Seq     int    `json:"seq"`               // 块序号，从 0 开始
	Total   int    `json:"total"`             // 总块数
	IsFinal bool   `json:"is_final"`          // 是否为最后一块
	Data    string `json:"data"`              // base64 编码的块内容
	Perm    uint32 `json:"perm,omitempty"`    // 文件权限（Unix），0 默认 0644
}

// WriteFileACKPayload 写文件分块确认（Local → Server）
type WriteFileACKPayload struct {
	Path      string `json:"path"`
	Seq       int    `json:"seq"`               // 当前确认的块序号
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
	BytesWritten int64 `json:"bytes_written,omitempty"` // 最终块时回传总写入字节数
}
