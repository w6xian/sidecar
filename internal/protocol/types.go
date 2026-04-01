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

	// MsgUpdate Server → Local：推送更新指令
	MsgUpdate MessageType = "UPDATE"
	// MsgUpdateACK Local → Server：更新指令响应
	MsgUpdateACK MessageType = "UPDATE_ACK"

	// MsgExecLua Server → Local：执行 Lua 脚本
	MsgExecLua MessageType = "EXEC_LUA"
	// MsgExecLuaResult Local → Server：Lua 脚本执行结果
	MsgExecLuaResult MessageType = "EXEC_LUA_RESULT"

	// MsgUploadFile Server → Local：请求上传文件（日志）
	MsgUploadFile MessageType = "UPLOAD_FILE"
	// MsgUploadFileData Local → Server：文件数据
	MsgUploadFileData MessageType = "UPLOAD_FILE_DATA"

	// MsgWriteFileChunk Server → Local：写文件分块（支持分段写入）
	MsgWriteFileChunk MessageType = "WRITE_FILE_CHUNK"
	// MsgWriteFileACK Local → Server：写文件分块确认
	MsgWriteFileACK MessageType = "WRITE_FILE_ACK"
)

// Protocol 定义服务协议类型
type Protocol string

const (
	ProtocolHTTP Protocol = "http"
	ProtocolGRPC Protocol = "grpc"
)
