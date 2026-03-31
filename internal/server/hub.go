package server

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/w6xian/sidecar/internal/protocol"
	"github.com/w6xian/sidecar/pkg/logger"
	"github.com/w6xian/sidecar/pkg/metrics"
)

// SidecarConn 代表一个已连接的 Sidecar 客户端
type SidecarConn struct {
	ID       string
	Alias    string // 连接别名（用于 /{alias}/* 路由）
	Conn     *websocket.Conn
	Services []protocol.ServiceInfo
	mu       sync.Mutex
	send     chan []byte

	// pendingRequests 存储等待响应的请求 (requestId → response channel)
	pendingRequests sync.Map
}

// NewSidecarConn 创建 Sidecar 连接实例
func NewSidecarConn(conn *websocket.Conn) *SidecarConn {
	return &SidecarConn{
		ID:   uuid.New().String(),
		Conn: conn,
		send: make(chan []byte, 256),
	}
}

// SendMessage 发送消息到 Sidecar
func (sc *SidecarConn) SendMessage(msg *protocol.Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	sc.send <- data
	return nil
}

// SendProxyRequest 发送代理请求并等待响应
func (sc *SidecarConn) SendProxyRequest(req *protocol.Message, timeout time.Duration) (*protocol.Message, error) {
	ch := make(chan *protocol.Message, 1)
	sc.pendingRequests.Store(req.RequestID, ch)
	defer sc.pendingRequests.Delete(req.RequestID)

	if err := sc.SendMessage(req); err != nil {
		return nil, err
	}

	select {
	case resp := <-ch:
		return resp, nil
	case <-time.After(timeout):
		return nil, ErrRequestTimeout
	}
}

// HandleResponse 处理来自 Sidecar 的响应
func (sc *SidecarConn) HandleResponse(msg *protocol.Message) {
	if v, ok := sc.pendingRequests.Load(msg.RequestID); ok {
		if ch, ok := v.(chan *protocol.Message); ok {
			select {
			case ch <- msg:
			default:
			}
		}
	}
}

// WritePump 写入泵，从 send channel 读取并写入 WebSocket
func (sc *SidecarConn) WritePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		sc.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-sc.send:
			sc.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				sc.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := sc.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				logger.L().Error("write message failed", zap.Error(err))
				return
			}
		case <-ticker.C:
			sc.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := sc.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Hub 管理所有已连接的 Sidecar
type Hub struct {
	// sidecars 存储所有连接 (connID → SidecarConn)
	sidecars sync.Map

	// aliases 存储 alias → connID 映射，用于 /{alias}/* 路由
	aliases sync.Map

	// register/unregister channels
	register   chan *SidecarConn
	unregister chan *SidecarConn

	// router 路由管理
	router *Router
}

// NewHub 创建连接管理器
func NewHub(router *Router) *Hub {
	return &Hub{
		register:   make(chan *SidecarConn),
		unregister: make(chan *SidecarConn),
		router:     router,
	}
}

// Run 启动 Hub
func (h *Hub) Run() {
	for {
		select {
		case conn := <-h.register:
			h.sidecars.Store(conn.ID, conn)
			if conn.Alias != "" {
				h.aliases.Store(conn.Alias, conn.ID)
			}
			metrics.ConnectedSidecars.Inc()
			logger.L().Info("sidecar connected",
				zap.String("conn_id", conn.ID),
				zap.String("alias", conn.Alias),
			)

		case conn := <-h.unregister:
			if _, loaded := h.sidecars.LoadAndDelete(conn.ID); loaded {
				if conn.Alias != "" {
					h.aliases.Delete(conn.Alias)
				}
				close(conn.send)
				// 从路由表中移除该连接注册的所有服务
				h.router.RemoveByConn(conn.ID)
				metrics.ConnectedSidecars.Dec()
				logger.L().Info("sidecar disconnected",
					zap.String("conn_id", conn.ID),
					zap.String("alias", conn.Alias),
				)
			}
		}
	}
}

// Register 注册一个新的 Sidecar 连接
func (h *Hub) Register(conn *SidecarConn) {
	h.register <- conn
}

// Unregister 注销一个 Sidecar 连接
func (h *Hub) Unregister(conn *SidecarConn) {
	h.unregister <- conn
}

// GetConn 获取指定连接
func (h *Hub) GetConn(connID string) (*SidecarConn, bool) {
	if v, ok := h.sidecars.Load(connID); ok {
		return v.(*SidecarConn), true
	}
	return nil, false
}

// GetConnByAlias 根据 alias 获取连接
func (h *Hub) GetConnByAlias(alias string) (*SidecarConn, bool) {
	if v, ok := h.aliases.Load(alias); ok {
		connID := v.(string)
		return h.GetConn(connID)
	}
	return nil, false
}

// AliasExists 检查 alias 是否已被占用
func (h *Hub) AliasExists(alias string) bool {
	if alias == "" {
		return false
	}
	_, ok := h.aliases.Load(alias)
	return ok
}
