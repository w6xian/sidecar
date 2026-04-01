package sidecar

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"go.uber.org/zap"

	"github.com/w6xian/sidecar/internal/protocol"
	"github.com/w6xian/sidecar/pkg/logger"
)

// writeFileSession 记录一次写文件会话的状态（支持并发多文件传输）
type writeFileSession struct {
	mu           sync.Mutex
	file         *os.File
	bytesWritten int64
	closed       bool
}

// writeFileSessions 管理 requestID → session 的映射
var (
	wfSessionsMu sync.Mutex
	wfSessions   = make(map[string]*writeFileSession)
)

// handleWriteFileChunk 处理服务端发来的写文件分块指令
func (c *Client) handleWriteFileChunk(msg *protocol.Message) {
	var payload protocol.WriteFileChunkPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		logger.L().Error("unmarshal write_file_chunk payload failed", zap.Error(err))
		c.sendWriteFileACK(msg.RequestID, payload.Path, payload.Seq, false, 0,
			"invalid payload: "+err.Error())
		return
	}

	logger.L().Info("received write_file_chunk",
		zap.String("path", payload.Path),
		zap.Int("seq", payload.Seq),
		zap.Int("total", payload.Total),
		zap.Bool("is_final", payload.IsFinal),
	)

	// 解码数据块
	chunk, err := base64.StdEncoding.DecodeString(payload.Data)
	if err != nil {
		logger.L().Error("decode chunk data failed", zap.Error(err))
		c.sendWriteFileACK(msg.RequestID, payload.Path, payload.Seq, false, 0,
			"base64 decode failed: "+err.Error())
		return
	}

	// 获取或创建会话
	wfSessionsMu.Lock()
	sess, exists := wfSessions[msg.RequestID]
	if !exists {
		sess = &writeFileSession{}
		wfSessions[msg.RequestID] = sess
	}
	wfSessionsMu.Unlock()

	sess.mu.Lock()
	defer sess.mu.Unlock()

	if sess.closed {
		// 已关闭的会话，忽略重复块
		return
	}

	// seq==0：创建文件（自动创建目录）
	if payload.Seq == 0 {
		dir := filepath.Dir(payload.Path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			logger.L().Error("mkdir failed", zap.String("dir", dir), zap.Error(err))
			c.sendWriteFileACK(msg.RequestID, payload.Path, payload.Seq, false, 0,
				"mkdir failed: "+err.Error())
			c.cleanWriteFileSession(msg.RequestID)
			return
		}

		perm := os.FileMode(0644)
		if payload.Perm != 0 {
			perm = os.FileMode(payload.Perm)
		}

		f, err := os.OpenFile(payload.Path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
		if err != nil {
			logger.L().Error("create file failed", zap.String("path", payload.Path), zap.Error(err))
			c.sendWriteFileACK(msg.RequestID, payload.Path, payload.Seq, false, 0,
				"create file failed: "+err.Error())
			c.cleanWriteFileSession(msg.RequestID)
			return
		}
		sess.file = f
		sess.bytesWritten = 0
	}

	// 写入数据块
	if sess.file == nil {
		c.sendWriteFileACK(msg.RequestID, payload.Path, payload.Seq, false, 0,
			"session file not initialized, missing seq=0 chunk")
		return
	}

	n, err := sess.file.Write(chunk)
	if err != nil {
		logger.L().Error("write chunk failed",
			zap.String("path", payload.Path),
			zap.Int("seq", payload.Seq),
			zap.Error(err),
		)
		c.sendWriteFileACK(msg.RequestID, payload.Path, payload.Seq, false, sess.bytesWritten,
			"write failed: "+err.Error())
		sess.file.Close()
		sess.closed = true
		c.cleanWriteFileSession(msg.RequestID)
		return
	}
	sess.bytesWritten += int64(n)

	// 最后一块：关闭文件，清理会话，回 ACK 含总字节数
	if payload.IsFinal {
		if err := sess.file.Close(); err != nil {
			logger.L().Error("close file failed", zap.String("path", payload.Path), zap.Error(err))
			c.sendWriteFileACK(msg.RequestID, payload.Path, payload.Seq, false, sess.bytesWritten,
				"close file failed: "+err.Error())
		} else {
			logger.L().Info("write file completed",
				zap.String("path", payload.Path),
				zap.Int64("bytes", sess.bytesWritten),
			)
			c.sendWriteFileACK(msg.RequestID, payload.Path, payload.Seq, true, sess.bytesWritten, "")
		}
		sess.closed = true
		c.cleanWriteFileSession(msg.RequestID)
		return
	}

	// 中间块：回确认，服务端收到后再发下一块
	c.sendWriteFileACK(msg.RequestID, payload.Path, payload.Seq, true, sess.bytesWritten, "")
}

// sendWriteFileACK 回写文件确认消息
func (c *Client) sendWriteFileACK(requestID, path string, seq int, success bool, bytesWritten int64, errMsg string) {
	ack := &protocol.WriteFileACKPayload{
		Path:         path,
		Seq:          seq,
		Success:      success,
		Error:        errMsg,
		BytesWritten: bytesWritten,
	}
	msg, err := protocol.NewMessage(protocol.MsgWriteFileACK, requestID, ack)
	if err != nil {
		return
	}
	raw, err := json.Marshal(msg)
	if err != nil {
		return
	}
	if err := c.writeMessage(raw); err != nil {
		logger.L().Error("send write_file_ack failed", zap.Error(err))
	}
}

// cleanWriteFileSession 清理写文件会话
func (c *Client) cleanWriteFileSession(requestID string) {
	wfSessionsMu.Lock()
	defer wfSessionsMu.Unlock()
	delete(wfSessions, requestID)
}
