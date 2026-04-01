package sidecar

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"go.uber.org/zap"

	"github.com/w6xian/sidecar/internal/protocol"
	"github.com/w6xian/sidecar/pkg/logger"
)

// handleUpdate 处理服务端下发的更新指令
func (c *Client) handleUpdate(msg *protocol.Message) {
	var payload protocol.UpdatePayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		logger.L().Error("unmarshal update payload failed", zap.Error(err))
		c.sendUpdateACK(msg.RequestID, false, "invalid payload: "+err.Error(), "")
		return
	}

	logger.L().Info("received update command",
		zap.String("version", payload.Version),
		zap.String("down_url", payload.DownURL),
		zap.Bool("force", payload.Force),
	)

	// 异步执行更新，避免阻塞消息循环
	go c.doUpdate(msg.RequestID, &payload)
}

// doUpdate 下载并替换可执行文件，然后重启
func (c *Client) doUpdate(requestID string, payload *protocol.UpdatePayload) {
	start := time.Now()

	// 1. 下载新版本
	logger.L().Info("downloading new version", zap.String("url", payload.DownURL))
	resp, err := http.Get(payload.DownURL)
	if err != nil {
		logger.L().Error("download failed", zap.Error(err))
		c.sendUpdateACK(requestID, false, "download failed: "+err.Error(), "")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.sendUpdateACK(requestID, false, fmt.Sprintf("download failed: HTTP %d", resp.StatusCode), "")
		return
	}

	// 2. 写入临时文件
	exePath, err := os.Executable()
	if err != nil {
		c.sendUpdateACK(requestID, false, "get executable path failed: "+err.Error(), "")
		return
	}
	exePath, _ = filepath.EvalSymlinks(exePath)

	tmpFile := exePath + ".new"
	f, err := os.OpenFile(tmpFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		c.sendUpdateACK(requestID, false, "create tmp file failed: "+err.Error(), "")
		return
	}

	h := sha256.New()
	w := io.MultiWriter(f, h)
	size, err := io.Copy(w, resp.Body)
	f.Close()
	if err != nil {
		os.Remove(tmpFile)
		c.sendUpdateACK(requestID, false, "write tmp file failed: "+err.Error(), "")
		return
	}

	logger.L().Info("download completed",
		zap.Int64("bytes", size),
		zap.Duration("elapsed", time.Since(start)),
	)

	// 3. 校验 checksum（可选）
	if payload.Checksum != "" {
		got := hex.EncodeToString(h.Sum(nil))
		if got != payload.Checksum {
			os.Remove(tmpFile)
			c.sendUpdateACK(requestID, false,
				fmt.Sprintf("checksum mismatch: expected %s got %s", payload.Checksum, got), "")
			return
		}
		logger.L().Info("checksum verified", zap.String("sha256", got))
	}

	// 4. 替换可执行文件（Windows 需要先备份）
	backupPath := exePath + ".bak"
	if err := os.Rename(exePath, backupPath); err != nil {
		os.Remove(tmpFile)
		c.sendUpdateACK(requestID, false, "backup old binary failed: "+err.Error(), "")
		return
	}
	if err := os.Rename(tmpFile, exePath); err != nil {
		// 回滚
		os.Rename(backupPath, exePath)
		c.sendUpdateACK(requestID, false, "replace binary failed: "+err.Error(), "")
		return
	}

	logger.L().Info("binary replaced, sending ACK and restarting",
		zap.String("version", payload.Version),
	)

	// 5. 回告服务端成功
	c.sendUpdateACK(requestID, true, "update applied, restarting", payload.Version)

	// 6. 重启自身
	time.Sleep(500 * time.Millisecond) // 等待 ACK 发出
	c.restartSelf(exePath)
}

// sendUpdateACK 发送更新响应
func (c *Client) sendUpdateACK(requestID string, success bool, message, version string) {
	ack := &protocol.UpdateACKPayload{
		Success: success,
		Message: message,
		Version: version,
	}
	msg, err := protocol.NewMessage(protocol.MsgUpdateACK, requestID, ack)
	if err != nil {
		return
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	if err := c.writeMessage(data); err != nil {
		logger.L().Error("send update ack failed", zap.Error(err))
	}
}

// restartSelf 重启当前进程
func (c *Client) restartSelf(exePath string) {
	logger.L().Info("restarting process", zap.String("path", exePath))

	args := os.Args[1:]
	cmd := exec.Command(exePath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if runtime.GOOS != "windows" {
		// Unix：exec 替换当前进程
		if err := cmd.Start(); err != nil {
			logger.L().Error("restart failed", zap.Error(err))
			return
		}
	} else {
		// Windows：start 子进程后退出
		if err := cmd.Start(); err != nil {
			logger.L().Error("restart failed", zap.Error(err))
			return
		}
	}

	// 停止当前客户端后退出
	c.Stop()
	os.Exit(0)
}
