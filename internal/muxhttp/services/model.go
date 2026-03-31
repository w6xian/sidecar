package services

import (
	"encoding/json"
	"fmt"

	"github.com/w6xian/sidecar/internal/crypto/aes"
	"github.com/w6xian/sidecar/internal/options"
	"github.com/w6xian/sidecar/internal/utils"
	"github.com/w6xian/sidecar/internal/utils/id"
)

type Service struct {
	Header  map[string]string
	Url     string
	Payload []byte
}

func Rpc2Url(url, m, f, appKey string) string {
	return fmt.Sprintf("%s/%s?s=%s&c=%s", url, f, m, appKey)
}

func NewRpc(url, m, f, appKey string) *Service {
	uri := Rpc2Url(url, m, f, appKey)
	return &Service{
		Url: uri,
	}
}

func (svr *Service) SetHeader(k, v string) {
	svr.Header[k] = v
}

func (svr *Service) SetBody(body, key []byte) error {
	if key == nil {
		svr.Payload = body
	}
	risk := options.RiskData{}
	risk.Id = id.GetUuid()
	risk.Time = utils.UnixTime()
	ciphertext, err := aes.Base64AESEBCEncrypt(body, key)
	if err != nil {
		return err
	}
	risk.Body = ciphertext
	riskData, err := json.Marshal(risk)
	if err != nil {
		return err
	}
	svr.Payload = riskData
	return nil
}
