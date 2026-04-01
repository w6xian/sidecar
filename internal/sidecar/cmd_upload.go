package sidecar

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"strings"

	"go.uber.org/zap"

	"github.com/w6xian/sidecar/internal/protocol"
	"github.com/w6xian/sidecar/pkg/logger"
)

const maxUploadBytes = 32 * 1024 * 1024 // 32MB 上限

// handleUploadFile 处理服务端发来的文件上传请求
func (c *Client) handleUploadFile(msg *protocol.Message) {
	var payload protocol.UploadFilePayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		logger.L().Error("unmarshal upload_file payload failed", zap.Error(err))
		c.sendUploadFileData(msg.RequestID, false, payload.Path, nil, 0, false, "invalid payload: "+err.Error())
		return
	}

	logger.L().Info("received upload_file request",
		zap.String("path", payload.Path),
		zap.Int64("max_bytes", payload.MaxBytes),
		zap.Int("tail", payload.Tail),
	)

	go c.doUploadFile(msg.RequestID, &payload)
}

// doUploadFile 读取本地文件并回传
func (c *Client) doUploadFile(requestID string, payload *protocol.UploadFilePayload) {
	// 打开文件
	f, err := os.Open(payload.Path)
	if err != nil {
		logger.L().Error("open file failed",
			zap.String("path", payload.Path),
			zap.Error(err),
		)
		c.sendUploadFileData(requestID, false, payload.Path, nil, 0, false, "open file failed: "+err.Error())
		return
	}
	defer f.Close()

	// 确定读取上限
	limit := payload.MaxBytes
	if limit <= 0 || limit > maxUploadBytes {
		limit = maxUploadBytes
	}

	var (
		data      []byte
		truncated bool
	)

	if payload.Tail > 0 {
		// 读取尾部 N 行
		data, err = readTailLines(f, payload.Tail, limit)
		if err != nil {
			c.sendUploadFileData(requestID, false, payload.Path, nil, 0, false, "read tail lines failed: "+err.Error())
			return
		}
		if int64(len(data)) >= limit {
			truncated = true
		}
	} else {
		// 读取全部（受 limit 约束）
		lr := io.LimitReader(f, limit+1) // 多读 1 字节以判断是否截断
		data, err = io.ReadAll(lr)
		if err != nil {
			c.sendUploadFileData(requestID, false, payload.Path, nil, 0, false, "read file failed: "+err.Error())
			return
		}
		if int64(len(data)) > limit {
			data = data[:limit]
			truncated = true
		}
	}

	logger.L().Info("file read completed",
		zap.String("path", payload.Path),
		zap.Int("bytes", len(data)),
		zap.Bool("truncated", truncated),
	)

	c.sendUploadFileData(requestID, true, payload.Path, data, int64(len(data)), truncated, "")
}

// sendUploadFileData 发送文件数据到服务端
func (c *Client) sendUploadFileData(requestID string, success bool, path string,
	data []byte, size int64, truncated bool, errMsg string) {

	result := &protocol.UploadFileDataPayload{
		Success:   success,
		Path:      path,
		Size:      size,
		Truncated: truncated,
		Error:     errMsg,
	}
	if len(data) > 0 {
		result.Data = base64.StdEncoding.EncodeToString(data)
	}

	msg, err := protocol.NewMessage(protocol.MsgUploadFileData, requestID, result)
	if err != nil {
		return
	}
	raw, err := json.Marshal(msg)
	if err != nil {
		return
	}
	if err := c.writeMessage(raw); err != nil {
		logger.L().Error("send upload_file_data failed", zap.Error(err))
	}
}

// readTailLines 读取文件末尾 n 行，最多 limit 字节
func readTailLines(f *os.File, n int, limit int64) ([]byte, error) {
	// 先全部扫描行，内存中保留最后 n 行（适合中小日志文件）
	scanner := bufio.NewScanner(io.LimitReader(f, limit))
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) > n {
			lines = lines[1:]
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return []byte(strings.Join(lines, "\n")), nil
}
