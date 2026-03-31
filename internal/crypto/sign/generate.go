package sign

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/url"
	"strconv"

	"IoT-printer/internal/utils"
)

func GenerateSignature(apiSecret, apiKey, timestamp, nonce string, params url.Values) string {
	message := apiKey + timestamp + nonce + params.Encode()
	mac := hmac.New(sha256.New, []byte(apiSecret))
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}

func DoQueryCheck(query url.Values, apiSecret string, timeoutSec int64) error {
	apiKey := query.Get("app_key")
	timestamp := query.Get("timestamp")
	nonce := query.Get("nonce")
	signature := query.Get("signature")
	// fmt.Println(apiKey, timestamp, nonce, signature, apiSecret)
	timeInt, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return err
	}
	if timeoutSec > 0 {
		if utils.UnixTime()-timeInt > timeoutSec {
			return errors.New("时间戳操时")
		}
	}
	query.Del("signature")
	expectedSignature := GenerateSignature(apiSecret, apiKey, timestamp, nonce, query)
	if hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return nil
	}
	return errors.New("签名错误")
}

func DoQuerySign(apiSecret, appKey, timestamp, nonceStr string, parmas url.Values) url.Values {
	parmas.Set("app_key", appKey)
	parmas.Set("nonce", nonceStr)
	parmas.Set("timestamp", timestamp)
	signature := GenerateSignature(apiSecret, appKey, timestamp, nonceStr, parmas)
	parmas.Set("signature", signature)
	// fmt.Println(appKey, timestamp, nonceStr, signature, apiSecret)
	return parmas
}
