package sidecar

import (
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/w6xian/sidecar/internal/crypto/aes"
	"github.com/w6xian/sidecar/internal/protocol"
	"github.com/w6xian/sidecar/pkg/logger"
)

// RegisterService 构建并发送服务注册消息
func RegisterService(client *Client) error {
	services := make([]protocol.ServiceInfo, 0, len(client.config.Services))
	for _, svc := range client.config.Services {
		services = append(services, protocol.ServiceInfo{
			ID:         svc.Id,
			Name:       svc.Name,
			Protocol:   protocol.Protocol(svc.Protocol),
			ExposePath: svc.ExposePath,
		})
	}

	regReq := &protocol.RegisterRequest{
		Services:  services,
		ConnAlias: client.config.Sidecar.ConnAlias,
	}

	msg := &protocol.Message{
		Type:      protocol.MsgRegister,
		AppId:     client.config.Sidecar.AppId,
		Timestamp: time.Now().Unix(),
	}
	// 3 校验 appSec 是否匹配
	appSec := fmt.Sprintf("%d-%s-%d", msg.Timestamp, client.config.Sidecar.AppSec, msg.Timestamp)
	// appSec 加密后与 sign 对比
	sign, err := aes.Base64AESEBCEncrypt([]byte(appSec), aes.GetAES256Key([]byte(client.config.Sidecar.AppSn)))
	msg.Sign = sign
	payload, err := protocol.EncodePayload(regReq)
	if err != nil {
		return err
	}
	msg.Payload = payload

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	if err := client.writeMessage(data); err != nil {
		return err
	}

	logger.L().Info("register request sent",
		zap.Int("service_count", len(services)),
		zap.String("conn_alias", client.config.Sidecar.ConnAlias),
	)

	return nil
}

// HandleRegisterACK 处理注册确认消息
func HandleRegisterACK(msg *protocol.Message) {
	var ack protocol.RegisterACK
	if err := json.Unmarshal(msg.Payload, &ack); err != nil {
		logger.L().Error("unmarshal register ack failed", zap.Error(err))
		return
	}

	if ack.Success {
		logger.L().Info("services registered successfully",
			zap.Strings("accepted", ack.Accepted),
			zap.String("assigned_alias", ack.ConnAlias),
		)
	} else {
		logger.L().Warn("service registration failed",
			zap.String("message", ack.Message),
		)
	}
}

// SendUnregister 发送服务注销消息
func SendUnregister(client *Client) error {
	serviceIDs := make([]string, 0, len(client.config.Services))
	for _, svc := range client.config.Services {
		serviceIDs = append(serviceIDs, svc.Id)
	}

	unreg := &protocol.UnregisterRequest{
		ServiceIDs: serviceIDs,
	}

	msg, err := protocol.NewMessage(protocol.MsgUnregister, "", unreg)
	if err != nil {
		return err
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return client.writeMessage(data)
}
