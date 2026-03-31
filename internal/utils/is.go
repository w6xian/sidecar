package utils

import (
	"os"
	"reflect"
	"regexp"

	"github.com/w6xian/sidecar/internal/utils/array"
)

func Nil(x any) bool {
	if x == nil {
		return true
	}
	return reflect.ValueOf(x).IsNil()
}

func ServiceArgs() bool {
	if len(os.Args) > 1 {
		arg := os.Args[1]
		return array.InArray(arg, []string{"install", "uninstall"})
	}
	return false
}

func ExistFile(filename string) bool {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return false
	}
	return true
}

// IsEmptyString 检查字符串是否为空
func EmptyString(s string) bool {
	return s == ""
}

// IsMobile 检查字符串是否为手机号
func Mobile(m string) bool {
	if len(m) != 11 {
		return false
	}
	// 正则表达式匹配中国手机号：
	// ^1[3-9]\d{9}$
	// 解释：
	// ^1        : 以1开头
	// [3-9]     : 第二位为3-9（排除1和2，对应最新号段规则）
	// \d{9}$    : 后面跟9位数字
	mobileRegex := regexp.MustCompile(`^1[3-9]\d{9}$`)
	return mobileRegex.MatchString(m)
}

func IsNumber(s string) bool {
	// 定义一个正则表达式，用于匹配数字
	reg := regexp.MustCompile(`^-?\d+(\.\d+)?$`)
	// 使用正则表达式进行匹配
	return reg.MatchString(s)
}

// 支付宝：https://docs.open.alipay.com/194/106039/
// 用户付款码，25-30 开头的长度为 16-24 位的数字，实际字符串长度以开发者获取的付款码长度为准；付款码使用一次即失效
// 281880908331692604  支付宝  282491369942753740
// 微信：https://pay.weixin.qq.com/wiki/doc/api/micropay.php?chapter=5_1
// 18位纯数字，以10、11、12、13、14、15开头[官方接口]
// 132687171730364633  微信

// 微信当面付

func IsWechatCode(code string) bool {
	if len(code) != 18 {
		return false
	}
	r := "^1[012345][0-9]{16}$"
	if m, err := regexp.Match(r, []byte(code)); err != nil {
		return false
	} else {
		return m
	}
}

// 支付宝当面付
func IsAlipayCode(code string) bool {
	r := "^[23][056789][0-9]{14,22}$"
	if m, err := regexp.Match(r, []byte(code)); err != nil {
		return false
	} else {
		return m
	}
}
