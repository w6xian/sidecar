package utils

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

func Orz3Decode(token string) string {
	if strings.HasPrefix(token, "orz:L/") || strings.HasPrefix(token, "L/") {
		s := strings.Split(token, "/")
		if len(s) == 3 {
			session := strings.TrimRight(s[1], "G")
			// ip, _ := strconv.ParseInt(strings.Split(s[2], ";")[0], 10, 64)
			// ipv4 := util.InetNtoA(ip)
			return session
		}
	}
	return token
}

func Orz3Encode(token string, ip string) string {
	return fmt.Sprintf("L/%s;no-cache", parseOrz3(token, ip))
}

func parseOrz3(token string, ip string) string {
	intIp := IPv4ToInt(ip)
	// !!! 取后8位，这里用服务器加密，其他软件要注意这里
	l8 := token[8:]
	k := GetInt(string(l8)) & 0xFFFF
	return fmt.Sprintf("%sG/%d", token, intIp^uint32(k))
}

func IsOrz(token string) bool {
	if strings.HasPrefix(token, "orz:L/") || strings.HasPrefix(token, "L/") {
		return true
	}
	return false
}

func GenerateSignature(apiKey, apiSecret string, timestamp int64, code, token string) string {
	message := fmt.Sprintf("%s%d%s%s", apiKey, timestamp, code, token)
	mac := hmac.New(sha256.New, []byte(apiSecret))
	mac.Write([]byte(message))
	signature := hex.EncodeToString(mac.Sum(nil))
	return signature
}
