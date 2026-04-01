package sidecar

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	"go.uber.org/zap"

	"github.com/w6xian/sidecar/internal/protocol"
	"github.com/w6xian/sidecar/pkg/logger"
)

// handleExecCmd 处理执行命令指令
func (c *Client) handleExecCmd(msg *protocol.Message) {
	var payload protocol.ExecCmdPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		logger.L().Error("unmarshal exec_cmd payload failed", zap.Error(err))
		c.sendExecCmdResult(msg.RequestID, false, "", "invalid payload: "+err.Error(), "", -1, 0)
		return
	}

	if payload.Command == "" {
		c.sendExecCmdResult(msg.RequestID, false, "", "command is required", "", -1, 0)
		return
	}

	// 异步执行，防止阻塞消息循环
	go c.doExecCmd(msg.RequestID, &payload)
}

// doExecCmd 执行命令并回传结果
func (c *Client) doExecCmd(requestID string, payload *protocol.ExecCmdPayload) {
	timeout := payload.Timeout
	if timeout <= 0 {
		timeout = 60
	}

	start := time.Now()

	cmd := buildExecCmd(payload)

	// 捕获 stdout 和 stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// 如果有 stdin 内容，写入管道
	if payload.Stdin != "" {
		stdinPipe, err := cmd.StdinPipe()
		if err != nil {
			c.sendExecCmdResult(requestID, false, "", "create stdin pipe failed: "+err.Error(), "", -1, 0)
			return
		}
		go func() {
			defer stdinPipe.Close()
			stdinPipe.Write([]byte(payload.Stdin))
		}()
	}

	// 启动进程（带超时控制）
	if err := cmd.Start(); err != nil {
		c.sendExecCmdResult(requestID, false, "", "start process failed: "+err.Error(), "", -1, time.Since(start).Milliseconds())
		return
	}

	// 等待进程完成或超时
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		elapsed := time.Since(start).Milliseconds()
		exitCode := 0
		if err != nil {
			exitCode = cmd.ProcessState.ExitCode()
			logger.L().Warn("exec command failed",
				zap.String("request_id", requestID),
				zap.String("command", payload.Command),
				zap.Int("exit_code", exitCode),
				zap.Error(err),
			)
		} else {
			logger.L().Info("exec command succeeded",
				zap.String("request_id", requestID),
				zap.String("command", payload.Command),
				zap.Int64("elapsed_ms", elapsed),
			)
		}

		c.sendExecCmdResult(requestID, err == nil, stdout.String(), stderr.String(), "", exitCode, elapsed)

	case <-time.After(time.Duration(timeout) * time.Second):
		// 超时，杀掉进程
		cmd.Process.Kill()
		<-done // 等待进程退出
		elapsed := time.Since(start).Milliseconds()
		logger.L().Warn("exec command timeout",
			zap.String("request_id", requestID),
			zap.String("command", payload.Command),
			zap.Int("timeout_s", timeout),
		)
		c.sendExecCmdResult(requestID, false, stdout.String(),
			fmt.Sprintf("execution timeout after %ds", timeout),
			"", -1, elapsed)
	}
}

// sendExecCmdResult 发送命令执行结果
func (c *Client) sendExecCmdResult(requestID string, success bool, output, errMsg, _ string, exitCode int, elapsed int64) {
	result := &protocol.ExecCmdResultPayload{
		Success:  success,
		Output:   output,
		Error:    errMsg,
		ExitCode: exitCode,
		Elapsed:  elapsed,
	}
	msg, err := protocol.NewMessage(protocol.MsgExecCmdResult, requestID, result)
	if err != nil {
		return
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	if err := c.writeMessage(data); err != nil {
		logger.L().Error("send exec_cmd result failed", zap.Error(err))
	}
}

// buildExecCmd 根据 ExecCmdPayload 构造 exec.Cmd
func buildExecCmd(payload *protocol.ExecCmdPayload) *exec.Cmd {
	cmd := exec.Command(payload.Command, payload.Args...)
	cmd.Dir = payload.Dir

	// 合并环境变量：先继承系统环境变量，再追加自定义
	if len(payload.Env) > 0 {
		cmd.Env = os.Environ()
		for k, v := range payload.Env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}

	return cmd
}
