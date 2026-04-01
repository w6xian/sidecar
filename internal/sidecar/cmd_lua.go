package sidecar

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/w6xian/gua"
	"github.com/w6xian/sidecar/internal/protocol"
	"github.com/w6xian/sidecar/pkg/logger"
)

// handleExecLua 处理执行 Lua 脚本指令
func (c *Client) handleExecLua(msg *protocol.Message) {
	var payload protocol.ExecLuaPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		logger.L().Error("unmarshal exec_lua payload failed", zap.Error(err))
		c.sendExecLuaResult(msg.RequestID, false, "", "invalid payload: "+err.Error(), 0)
		return
	}

	// 异步执行，防止阻塞消息循环
	go c.doExecLua(msg.RequestID, &payload)
}

// doExecLua 执行 Lua 脚本并回传结果
func (c *Client) doExecLua(requestID string, payload *protocol.ExecLuaPayload) {
	timeout := payload.Timeout
	if timeout <= 0 {
		timeout = 30
	}

	start := time.Now()

	// 捕获 print 输出
	var outputBuf bytes.Buffer

	// 创建 Lua 状态机
	L := gua.NewState()
	defer L.Close()

	// 注册 print 替换函数，捕获输出
	capture := &luaOutputCapture{buf: &outputBuf}
	L.SetGlobal(capture)

	// 注入环境变量作为全局 Lua 变量（字符串类型）
	for k, v := range payload.Env {
		// 用 DoString 注入简单字符串全局变量
		escaped := strings.ReplaceAll(v, `\`, `\\`)
		escaped = strings.ReplaceAll(escaped, `"`, `\"`)
		L.DoString(fmt.Sprintf(`%s = "%s"`, k, escaped))
	}

	// 执行脚本
	var execErr error
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() {
			if r := recover(); r != nil {
				execErr = fmt.Errorf("lua panic: %v", r)
			}
		}()

		if payload.Script != "" {
			L.DoString(payload.Script)
		} else if payload.File != "" {
			L.DoFile(payload.File)
		} else {
			execErr = fmt.Errorf("neither script nor file specified")
		}
	}()

	// 超时控制
	select {
	case <-done:
	case <-time.After(time.Duration(timeout) * time.Second):
		execErr = fmt.Errorf("execution timeout after %ds", timeout)
	}

	elapsed := time.Since(start).Milliseconds()
	output := outputBuf.String()

	if execErr != nil {
		logger.L().Warn("lua script execution failed",
			zap.String("request_id", requestID),
			zap.Error(execErr),
		)
		c.sendExecLuaResult(requestID, false, output, execErr.Error(), elapsed)
		return
	}

	logger.L().Info("lua script executed",
		zap.String("request_id", requestID),
		zap.Int64("elapsed_ms", elapsed),
	)
	c.sendExecLuaResult(requestID, true, output, "", elapsed)
}

// sendExecLuaResult 发送 Lua 执行结果
func (c *Client) sendExecLuaResult(requestID string, success bool, output, errMsg string, elapsed int64) {
	result := &protocol.ExecLuaResultPayload{
		Success: success,
		Output:  output,
		Error:   errMsg,
		Elapsed: elapsed,
	}
	msg, err := protocol.NewMessage(protocol.MsgExecLuaResult, requestID, result)
	if err != nil {
		return
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	if err := c.writeMessage(data); err != nil {
		logger.L().Error("send exec_lua result failed", zap.Error(err))
	}
}

// luaOutputCapture 捕获 Lua print 输出的辅助结构体
// 通过 SetGlobal 注册后，Lua 中可以调用 Print(...) 写入缓冲区
type luaOutputCapture struct {
	buf *bytes.Buffer
}

// Print 替代 Lua 的 print 函数，写入内部缓冲区
func (lc *luaOutputCapture) Print(args ...string) {
	lc.buf.WriteString(strings.Join(args, "\t"))
	lc.buf.WriteByte('\n')
}

// Log 辅助日志函数，也写入缓冲区
func (lc *luaOutputCapture) Log(msg string) {
	lc.buf.WriteString(msg)
	lc.buf.WriteByte('\n')
}
