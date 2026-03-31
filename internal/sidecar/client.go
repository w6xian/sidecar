package sidecar

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/w6xian/sidecar/internal/config"
	"github.com/w6xian/sidecar/internal/crypto/aes"
	"github.com/w6xian/sidecar/internal/protocol"
	"github.com/w6xian/sidecar/pkg/logger"
	"github.com/w6xian/sidecar/pkg/metrics"
)

// Client WebSocket 客户端
type Client struct {
	config    *config.Profile
	conn      *websocket.Conn
	mu        sync.Mutex
	forwarder *Forwarder
	done      chan struct{}
	connected bool
}

// NewClient 创建 Sidecar 客户端
func NewClient(config *config.Profile) *Client {
	return &Client{
		config:    config,
		forwarder: NewForwarder(config.Services),
		done:      make(chan struct{}),
	}
}

// Start 启动客户端（连接、注册、消息循环）
func (c *Client) Start() error {
	// 连接 Sidecar 服务器
	logger.L().Info("sidecar client connecting",
		zap.String("server", c.config.Sidecar.ServerAddr),
		zap.Int("service_count", len(c.config.Services)),
	)

	return c.connectWithRetry()
}

// Stop 停止客户端（优雅退出）
func (c *Client) Stop() {
	logger.L().Info("sidecar client stopping...")

	// 发送 UNREGISTER 消息
	if c.connected {
		if err := SendUnregister(c); err != nil {
			logger.L().Error("send unregister failed", zap.Error(err))
		}

		c.mu.Lock()
		if c.conn != nil {
			c.conn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, "shutdown"))
			c.conn.Close()
		}
		c.mu.Unlock()
	}

	close(c.done)
	logger.L().Info("sidecar client stopped")
}

// connectWithRetry 指数退避重连
func (c *Client) connectWithRetry() error {
	attempt := 0
	for {
		select {
		case <-c.done:
			return nil
		default:
		}

		if err := c.connect(); err != nil {
			attempt++
			delay := c.calcBackoff(attempt)
			logger.L().Warn("connection failed, retrying",
				zap.Error(err),
				zap.Int("attempt", attempt),
				zap.Duration("delay", delay),
			)
			metrics.WebSocketErrors.With(map[string]string{"type": "connect_failed"}).Inc()

			select {
			case <-time.After(delay):
				continue
			case <-c.done:
				return nil
			}
		}

		// 连接成功，进入消息循环
		attempt = 0
		c.connected = true
		c.readLoop()
		c.connected = false
		c.mu.Lock()
		if c.conn != nil {
			c.conn.Close()
			c.conn = nil
		}
		c.mu.Unlock()

		logger.L().Warn("connection lost, will reconnect")
	}
}

// connect 建立 WebSocket 连接并注册
func (c *Client) connect() error {
	header := http.Header{}
	t := time.Now().Unix()
	appSec := fmt.Sprintf("%d-%s-%d", t, c.config.Sidecar.AppSec, t)
	sign, err := aes.Base64AESEBCEncrypt([]byte(appSec), aes.GetAES256Key([]byte(c.config.Sidecar.AppSn)))
	header.Set("Authorization", fmt.Sprintf("Bearer %d,%s", t, sign))

	dialer := websocket.Dialer{
		HandshakeTimeout: 15 * time.Second,
	}

	conn, _, err := dialer.Dial(c.config.Sidecar.ServerAddr, header)
	if err != nil {
		return err
	}

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	cleanup := func() {
		c.mu.Lock()
		if c.conn == conn {
			c.conn = nil
		}
		c.mu.Unlock()
		conn.Close()
	}

	// 发送注册请求
	if err := RegisterService(c); err != nil {
		cleanup()
		return err
	}

	conn.SetReadDeadline(time.Now().Add(15 * time.Second))
	_, data, err := conn.ReadMessage()
	conn.SetReadDeadline(time.Time{})
	if err != nil {
		cleanup()
		return err
	}

	var msg protocol.Message
	if err := json.Unmarshal(data, &msg); err != nil {
		cleanup()
		return err
	}

	switch msg.Type {
	case protocol.MsgRegisterACK:
		HandleRegisterACK(&msg)
	case protocol.MsgError:
		var errPayload protocol.ErrorPayload
		if err := json.Unmarshal(msg.Payload, &errPayload); err == nil {
			cleanup()
			return fmt.Errorf("register rejected: %d %s", errPayload.Code, errPayload.Message)
		}
		cleanup()
		return errors.New("register rejected")
	default:
		cleanup()
		return fmt.Errorf("unexpected register response type: %s", msg.Type)
	}

	logger.L().Info("connected to server",
		zap.String("addr", c.config.Sidecar.ServerAddr),
	)

	return nil
}

// readLoop 消息读取循环
func (c *Client) readLoop() {
	c.conn.SetReadLimit(32 * 1024 * 1024) // 32MB

	// 心跳发送
	go c.heartbeatLoop()

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure,
			) {
				logger.L().Error("websocket read error", zap.Error(err))
				metrics.WebSocketErrors.With(map[string]string{"type": "read_error"}).Inc()
			}
			return
		}

		var msg protocol.Message
		if err := json.Unmarshal(data, &msg); err != nil {
			logger.L().Error("unmarshal message failed", zap.Error(err))
			continue
		}

		// 并发处理消息
		go c.handleMessage(&msg)
	}
}

// handleMessage 处理来自服务端的消息
func (c *Client) handleMessage(msg *protocol.Message) {
	switch msg.Type {
	case protocol.MsgRegisterACK:
		HandleRegisterACK(msg)

	case protocol.MsgProxyRequest:
		c.handleProxyRequest(msg)

	case protocol.MsgHeartbeat:
		metrics.HeartbeatTotal.With(map[string]string{"direction": "recv"}).Inc()

	case protocol.MsgError:
		var errPayload protocol.ErrorPayload
		if err := json.Unmarshal(msg.Payload, &errPayload); err == nil {
			logger.L().Error("received error from server",
				zap.Int("code", errPayload.Code),
				zap.String("message", errPayload.Message),
			)
		}

	default:
		logger.L().Warn("unknown message type",
			zap.String("type", string(msg.Type)),
		)
	}
}

// handleProxyRequest 处理代理请求
func (c *Client) handleProxyRequest(msg *protocol.Message) {
	var proxyReq protocol.ProxyRequest
	if err := json.Unmarshal(msg.Payload, &proxyReq); err != nil {
		logger.L().Error("unmarshal proxy request failed", zap.Error(err))
		c.sendError(msg.RequestID, 400, "invalid proxy request")
		return
	}

	logger.L().Debug("forwarding proxy request",
		zap.String("request_id", msg.RequestID),
		zap.String("service_id", proxyReq.ServiceID),
		zap.String("protocol", string(proxyReq.Protocol)),
	)

	// 转发到本地服务
	resp, err := c.forwarder.Forward(&proxyReq)
	if err != nil {
		logger.L().Error("forward failed",
			zap.String("request_id", msg.RequestID),
			zap.Error(err),
		)
		c.sendError(msg.RequestID, 502, "forward failed: "+err.Error())
		return
	}

	// 构造并发送响应
	respMsg, err := protocol.NewMessage(protocol.MsgProxyResponse, msg.RequestID, resp)
	if err != nil {
		logger.L().Error("encode response failed", zap.Error(err))
		c.sendError(msg.RequestID, 500, "encode response failed")
		return
	}

	data, err := json.Marshal(respMsg)
	if err != nil {
		logger.L().Error("marshal response failed", zap.Error(err))
		return
	}

	if err := c.writeMessage(data); err != nil {
		logger.L().Error("send response failed",
			zap.String("request_id", msg.RequestID),
			zap.Error(err),
		)
	}
}

// heartbeatLoop 心跳发送循环
func (c *Client) heartbeatLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			hb := &protocol.HeartbeatPayload{
				Timestamp: time.Now().UnixMilli(),
			}
			msg, _ := protocol.NewMessage(protocol.MsgHeartbeat, "", hb)
			data, _ := json.Marshal(msg)
			if err := c.writeMessage(data); err != nil {
				logger.L().Error("send heartbeat failed", zap.Error(err))
				return
			}
			metrics.HeartbeatTotal.With(map[string]string{"direction": "send"}).Inc()
		case <-c.done:
			return
		}
	}
}

// writeMessage 线程安全地写入 WebSocket 消息
func (c *Client) writeMessage(data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return nil
	}
	c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return c.conn.WriteMessage(websocket.TextMessage, data)
}

// sendError 发送错误消息到服务端
func (c *Client) sendError(requestID string, code int, message string) {
	errMsg, err := protocol.NewErrorMessage(requestID, code, message)
	if err != nil {
		return
	}
	data, err := json.Marshal(errMsg)
	if err != nil {
		return
	}
	c.writeMessage(data)
}

// calcBackoff 计算指数退避延迟
func (c *Client) calcBackoff(attempt int) time.Duration {
	base := c.config.Sidecar.ReconnectInterval
	if base == 0 {
		base = 5 * time.Second
	}
	max := c.config.Sidecar.MaxReconnectInterval
	if max == 0 {
		max = 60 * time.Second
	}

	delay := base * time.Duration(math.Pow(2, float64(attempt-1)))
	if delay > max {
		delay = max
	}
	return delay
}
