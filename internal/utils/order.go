package utils

import (
	"strings"
)

func GetPayTypeInt(payType string) int64 {
	switch strings.ToUpper(payType) {
	case "BALANCE":
		return 1
	case "CASH":
		return 2
	case "WECHAT":
		return 3
	case "ALIPAY":
		return 4
	case "UNIONPAY":
		return 5
	case "OTHER":
		return 6
	case "CREDIT":
		return 7
	case "RECORD":
		return 8
	case "MICROPAY":
		return 9
	case "JDPAY":
		return 10
	case "QQPAY":
		return 11
	default:
		return 99
	}
}
