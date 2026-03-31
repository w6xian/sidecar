package protocol

// MessageType 定义 WebSocket 消息类型
type MessageType string

const (
	// MsgRegister Local → Server：注册服务
	MsgRegister MessageType = "REGISTER"
	// MsgRegisterACK Server → Local：注册确认
	MsgRegisterACK MessageType = "REGISTER_ACK"
	// MsgProxyRequest Server → Local：转发请求
	MsgProxyRequest MessageType = "PROXY_REQUEST"
	// MsgProxyResponse Local → Server：转发响应
	MsgProxyResponse MessageType = "PROXY_RESPONSE"
	// MsgHeartbeat 双向心跳
	MsgHeartbeat MessageType = "HEARTBEAT"
	// MsgUnregister Local → Server：注销服务
	MsgUnregister MessageType = "UNREGISTER"
	// MsgError 双向错误通知
	MsgError MessageType = "ERROR"
)

// Protocol 定义服务协议类型
type Protocol string

const (
	ProtocolHTTP Protocol = "http"
	ProtocolGRPC Protocol = "grpc"
)
