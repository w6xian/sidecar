package server

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"


	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/w6xian/sidecar/internal/protocol"
	"github.com/w6xian/sidecar/internal/store"
	"github.com/w6xian/sidecar/pkg/auth"
	"github.com/w6xian/sidecar/pkg/logger"
	"github.com/w6xian/sidecar/pkg/metrics"
)

var (
	ErrRequestTimeout = errors.New("proxy request timeout")
	ErrNoSidecar      = errors.New("no available sidecar for service")
	ErrServiceUnknown = errors.New("unknown service")
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 生产环境应限制 Origin
	},
}

// Handler WebSocket 连接处理器
type Handler struct {
	hub          *Hub
	router       *Router
	auth         auth.Authenticator
	ipWhitelist  *auth.IPWhitelist
	proxyTimeout time.Duration
	Store        *store.Store
	Logger       *zap.Logger
}

// NewHandler 创建 Handler
func NewHandler(hub *Hub, router *Router, a auth.Authenticator, ipWhitelist *auth.IPWhitelist, proxyTimeout time.Duration) *Handler {
	return &Handler{
		hub:          hub,
		router:       router,
		auth:         a,
		ipWhitelist:  ipWhitelist,
		proxyTimeout: proxyTimeout,
	}
}

// ServeWS 处理 WebSocket 升级请求
func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
	// IP 白名单检查
	clientIP := getClientIP(r)
	if !h.ipWhitelist.IsAllowed(clientIP) {
		http.Error(w, "forbidden", http.StatusForbidden)
		logger.L().Warn("ip blocked", zap.String("ip", clientIP))
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.L().Error("websocket upgrade failed", zap.Error(err))
		return
	}

	sc := NewSidecarConn(conn)

	// 等待第一条消息：必须是 REGISTER 消息携带 Token
	conn.SetReadDeadline(time.Now().Add(15 * time.Second))
	_, msgData, err := conn.ReadMessage()
	if err != nil {
		logger.L().Error("read first message failed", zap.Error(err))
		conn.Close()
		return
	}
	conn.SetReadDeadline(time.Time{})

	var firstMsg protocol.Message
	if err := json.Unmarshal(msgData, &firstMsg); err != nil {
		logger.L().Error("unmarshal first message failed", zap.Error(err))
		conn.Close()
		return
	}

	if firstMsg.Type != protocol.MsgRegister {
		logger.L().Warn("first message is not REGISTER",
			zap.String("type", string(firstMsg.Type)),
		)
		conn.Close()
		return
	}
	fmt.Println(firstMsg.AppId, firstMsg.Sign, firstMsg.Timestamp, firstMsg.Type)
	// 验证 Token
	token, err := h.auth.Validate(firstMsg.AppId, firstMsg.Sign, firstMsg.Timestamp)
	if err != nil {
		logger.L().Warn("auth failed", zap.Error(err), zap.String("ip", clientIP))
		errMsg, _ := protocol.NewErrorMessage("", 401, "authentication failed")
		data, _ := json.Marshal(errMsg)
		conn.WriteMessage(websocket.TextMessage, data)
		conn.Close()
		return
	}

	// 解析注册信息
	var regReq protocol.RegisterRequest
	if err := json.Unmarshal(firstMsg.Payload, &regReq); err != nil {
		logger.L().Error("unmarshal register request failed", zap.Error(err))
		conn.Close()
		return
	}

	// 处理连接别名
	actualAlias := token.Name
	if actualAlias != "" {
		// 检查 alias 是否已被占用
		if h.hub.AliasExists(actualAlias) {
			logger.L().Warn("alias already exists, using random UUID instead",
				zap.String("requested_alias", actualAlias),
			)
			actualAlias = "" // 将使用随机 UUID 作为别名
		}
	}
	if actualAlias == "" {
		// 未指定或冲突时使用连接 ID 作为别名
		actualAlias = sc.ID
	}
	sc.Alias = actualAlias

	// 注册服务到路由表
	accepted := h.router.Register(sc.ID, regReq.Services)
	sc.Services = regReq.Services

	// 发送注册确认
	ackPayload := &protocol.RegisterACK{
		Success:   true,
		Message:   "registered successfully",
		Accepted:  accepted,
		ConnAlias: actualAlias, // 告知客户端实际使用的别名
	}
	ackMsg, _ := protocol.NewMessage(protocol.MsgRegisterACK, "", ackPayload)
	ackData, _ := json.Marshal(ackMsg)
	conn.WriteMessage(websocket.TextMessage, ackData)

	// 注册到 Hub
	h.hub.Register(sc)

	// 启动写泵
	go sc.WritePump()

	// 读循环（主循环）
	h.readPump(sc)
}

// readPump 读取来自 Sidecar 的消息
func (h *Handler) readPump(sc *SidecarConn) {
	defer func() {
		h.hub.Unregister(sc)
	}()

	sc.Conn.SetReadLimit(32 * 1024 * 1024) // 32MB
	sc.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	sc.Conn.SetPongHandler(func(string) error {
		sc.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, data, err := sc.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure,
			) {
				logger.L().Error("websocket unexpected close",
					zap.String("conn_id", sc.ID),
					zap.Error(err),
				)
				metrics.WebSocketErrors.With(map[string]string{"type": "unexpected_close"}).Inc()
			}
			return
		}
		sc.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		var msg protocol.Message
		if err := json.Unmarshal(data, &msg); err != nil {
			logger.L().Error("unmarshal message failed",
				zap.String("conn_id", sc.ID),
				zap.Error(err),
			)
			continue
		}

		h.handleMessage(sc, &msg)
	}
}

// handleMessage 处理来自 Sidecar 的消息
func (h *Handler) handleMessage(sc *SidecarConn, msg *protocol.Message) {
	switch msg.Type {
	case protocol.MsgProxyResponse:
		sc.HandleResponse(msg)

	case protocol.MsgUpdateACK, protocol.MsgExecLuaResult, protocol.MsgUploadFileData, protocol.MsgWriteFileACK, protocol.MsgExecCmdResult:
		// 将响应回填到等待的请求通道
		sc.HandleResponse(msg)

	case protocol.MsgHeartbeat:
		metrics.HeartbeatTotal.With(map[string]string{"direction": "recv"}).Inc()
		// 回复心跳
		resp, _ := protocol.NewMessage(protocol.MsgHeartbeat, "", nil)
		sc.SendMessage(resp)
		metrics.HeartbeatTotal.With(map[string]string{"direction": "send"}).Inc()

	case protocol.MsgUnregister:
		var unreg protocol.UnregisterRequest
		if err := json.Unmarshal(msg.Payload, &unreg); err != nil {
			logger.L().Error("unmarshal unregister failed", zap.Error(err))
			return
		}
		h.router.Unregister(sc.ID, unreg.ServiceIDs)
		logger.L().Info("services unregistered",
			zap.String("conn_id", sc.ID),
			zap.Strings("service_ids", unreg.ServiceIDs),
		)

	case protocol.MsgError:
		logger.L().Warn("received error from sidecar",
			zap.String("conn_id", sc.ID),
			zap.String("payload", string(msg.Payload)),
		)
		// 尝试将错误传递给等待的请求
		sc.HandleResponse(msg)

	default:
		logger.L().Warn("unknown message type",
			zap.String("type", string(msg.Type)),
			zap.String("conn_id", sc.ID),
		)
	}
}

// ForwardHTTP 将 HTTP 请求转发到 Sidecar
func (h *Handler) ForwardHTTP(serviceID string, reqPayload *protocol.ProxyRequest) (*protocol.ProxyResponse, error) {
	entry, exists := h.router.MatchByServiceID(serviceID)
	if !exists {
		return nil, ErrServiceUnknown
	}

	connID := h.router.PickConn(entry)
	if connID == "" {
		return nil, ErrNoSidecar
	}

	sc, ok := h.hub.GetConn(connID)
	if !ok {
		return nil, ErrNoSidecar
	}

	requestID := uuid.New().String()
	reqMsg, err := protocol.NewMessage(protocol.MsgProxyRequest, requestID, reqPayload)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	respMsg, err := sc.SendProxyRequest(reqMsg, h.proxyTimeout)
	if err != nil {
		metrics.ProxyRequestsTotal.With(map[string]string{
			"service_id": serviceID,
			"protocol":   "http",
			"method":     reqPayload.HTTP.Method,
			"status":     "error",
		}).Inc()
		return nil, err
	}

	duration := time.Since(start).Seconds()
	metrics.ProxyRequestDuration.With(map[string]string{
		"service_id": serviceID,
		"protocol":   "http",
	}).Observe(duration)

	var resp protocol.ProxyResponse
	if err := json.Unmarshal(respMsg.Payload, &resp); err != nil {
		return nil, err
	}

	statusStr := "success"
	if resp.HTTP != nil && resp.HTTP.StatusCode >= 400 {
		statusStr = "error"
	}
	metrics.ProxyRequestsTotal.With(map[string]string{
		"service_id": serviceID,
		"protocol":   "http",
		"method":     reqPayload.HTTP.Method,
		"status":     statusStr,
	}).Inc()

	return &resp, nil
}

// SendUpdate 向指定 Sidecar 连接发送更新指令，等待 ACK
func (h *Handler) SendUpdate(connID string, payload *protocol.UpdatePayload) (*protocol.UpdateACKPayload, error) {
	sc, ok := h.hub.GetConn(connID)
	if !ok {
		return nil, ErrNoSidecar
	}

	requestID := uuid.New().String()
	msg, err := protocol.NewMessage(protocol.MsgUpdate, requestID, payload)
	if err != nil {
		return nil, err
	}

	respMsg, err := sc.SendProxyRequest(msg, h.proxyTimeout)
	if err != nil {
		return nil, err
	}

	var ack protocol.UpdateACKPayload
	if err := json.Unmarshal(respMsg.Payload, &ack); err != nil {
		return nil, err
	}
	return &ack, nil
}

// SendExecLua 向指定 Sidecar 连接发送执行 Lua 脚本指令，等待结果
func (h *Handler) SendExecLua(connID string, payload *protocol.ExecLuaPayload) (*protocol.ExecLuaResultPayload, error) {
	sc, ok := h.hub.GetConn(connID)
	if !ok {
		return nil, ErrNoSidecar
	}

	requestID := uuid.New().String()
	msg, err := protocol.NewMessage(protocol.MsgExecLua, requestID, payload)
	if err != nil {
		return nil, err
	}

	timeout := h.proxyTimeout
	if payload.Timeout > 0 {
		timeout = time.Duration(payload.Timeout+5) * time.Second // 留 5s buffer
	}

	respMsg, err := sc.SendProxyRequest(msg, timeout)
	if err != nil {
		return nil, err
	}

	var result protocol.ExecLuaResultPayload
	if err := json.Unmarshal(respMsg.Payload, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SendUploadFile 向指定 Sidecar 连接发送文件上传请求，等待数据
func (h *Handler) SendUploadFile(connID string, payload *protocol.UploadFilePayload) (*protocol.UploadFileDataPayload, error) {
	sc, ok := h.hub.GetConn(connID)
	if !ok {
		return nil, ErrNoSidecar
	}

	requestID := uuid.New().String()
	msg, err := protocol.NewMessage(protocol.MsgUploadFile, requestID, payload)
	if err != nil {
		return nil, err
	}

	respMsg, err := sc.SendProxyRequest(msg, h.proxyTimeout)
	if err != nil {
		return nil, err
	}

	var result protocol.UploadFileDataPayload
	if err := json.Unmarshal(respMsg.Payload, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SendExecCmd 向指定 Sidecar 连接发送执行命令指令，等待结果
func (h *Handler) SendExecCmd(connID string, payload *protocol.ExecCmdPayload) (*protocol.ExecCmdResultPayload, error) {
	sc, ok := h.hub.GetConn(connID)
	if !ok {
		return nil, ErrNoSidecar
	}

	requestID := uuid.New().String()
	msg, err := protocol.NewMessage(protocol.MsgExecCmd, requestID, payload)
	if err != nil {
		return nil, err
	}

	timeout := h.proxyTimeout
	if payload.Timeout > 0 {
		timeout = time.Duration(payload.Timeout+5) * time.Second // 留 5s buffer
	}

	respMsg, err := sc.SendProxyRequest(msg, timeout)
	if err != nil {
		return nil, err
	}

	var result protocol.ExecCmdResultPayload
	if err := json.Unmarshal(respMsg.Payload, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// WriteFileRequest 写文件请求参数（供 Gateway/HTTP 层使用）
type WriteFileRequest struct {
	Path    string `json:"path"`              // 目标文件路径（客户端本地）
	Data    string `json:"data"`              // base64 编码的完整文件内容
	Perm    uint32 `json:"perm,omitempty"`    // 文件权限，0 默认 0644
	ChunkSize int  `json:"chunk_size,omitempty"` // 每块字节数，0 默认 256KB
}

// WriteFileResult 写文件操作结果
type WriteFileResult struct {
	Success      bool   `json:"success"`
	Path         string `json:"path"`
	BytesWritten int64  `json:"bytes_written,omitempty"`
	Chunks       int    `json:"chunks"`
	Error        string `json:"error,omitempty"`
}

const defaultChunkSize = 256 * 1024 // 256KB

// SendWriteFile 向指定 Sidecar 分块写入文件
// 每块发送后同步等待 ACK，全部完成后返回汇总结果
func (h *Handler) SendWriteFile(connID string, req *WriteFileRequest) (*WriteFileResult, error) {
	sc, ok := h.hub.GetConn(connID)
	if !ok {
		return nil, ErrNoSidecar
	}

	// 解码完整文件内容
	fileData, err := base64.StdEncoding.DecodeString(req.Data)
	if err != nil {
		return nil, fmt.Errorf("decode data failed: %w", err)
	}

	chunkSize := req.ChunkSize
	if chunkSize <= 0 {
		chunkSize = defaultChunkSize
	}

	// 计算总块数
	total := (len(fileData) + chunkSize - 1) / chunkSize
	if total == 0 {
		total = 1 // 空文件也发一块（空内容）
	}

	requestID := uuid.New().String()
	var totalWritten int64

	for seq := 0; seq < total; seq++ {
		start := seq * chunkSize
		end := start + chunkSize
		if end > len(fileData) {
			end = len(fileData)
		}
		chunkBytes := fileData[start:end]

		isFinal := seq == total-1

		chunkPayload := &protocol.WriteFileChunkPayload{
			Path:    req.Path,
			Seq:     seq,
			Total:   total,
			IsFinal: isFinal,
			Data:    base64.StdEncoding.EncodeToString(chunkBytes),
			Perm:    req.Perm,
		}

		msg, err := protocol.NewMessage(protocol.MsgWriteFileChunk, requestID, chunkPayload)
		if err != nil {
			return nil, fmt.Errorf("build chunk msg (seq=%d) failed: %w", seq, err)
		}

		// 每块单独等 ACK（串行保序）
		respMsg, err := sc.SendProxyRequest(msg, h.proxyTimeout)
		if err != nil {
			return &WriteFileResult{
				Success: false,
				Path:    req.Path,
				Chunks:  seq,
				Error:   fmt.Sprintf("chunk %d: wait ack timeout/error: %v", seq, err),
			}, nil
		}

		var ack protocol.WriteFileACKPayload
		if err := json.Unmarshal(respMsg.Payload, &ack); err != nil {
			return nil, fmt.Errorf("unmarshal ack (seq=%d) failed: %w", seq, err)
		}
		if !ack.Success {
			return &WriteFileResult{
				Success: false,
				Path:    req.Path,
				Chunks:  seq,
				Error:   fmt.Sprintf("chunk %d rejected: %s", seq, ack.Error),
			}, nil
		}

		if isFinal {
			totalWritten = ack.BytesWritten
		}
	}

	return &WriteFileResult{
		Success:      true,
		Path:         req.Path,
		BytesWritten: totalWritten,
		Chunks:       total,
	}, nil
}

// getClientIP 获取客户端真实 IP

func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx >= 0 {
		ip = ip[:idx]
	}
	return ip
}
