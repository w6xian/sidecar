package utils

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/w6xian/sidecar/internal/number"
)

func ToNumber(str string) float64 {
	// 转换为浮点数
	var num float64
	fmt.Sscanf(str, "%f", &num)
	return num
}

func MarshalString(v any, def string) string {
	b, err := json.Marshal(v)
	if err != nil {
		return def
	}
	return string(b)
}

func ToString(v any) string {
	return fmt.Sprintf("%v", v)
}

func MaxString(str string, max int) string {
	if max == 0 {
		return str
	}
	var start, cnt int
	if max > 0 {
		for start = range str {
			if cnt == max {
				break
			}
			cnt++
		}
		return str[:start]
	}

	for start = range str {
		cnt++
	}
	if cnt+max <= 0 {
		return str
	}
	return str[cnt+max:]

}

// sql格式化 根据数据类型替换"?", 字符串类型会自动添加单引号, 数字类型直接替换
// 示例: SqlFilter("select * from table where id = ? and name = ?", 1, "张三")
// 结果: select * from table where id = 1 and name = '张三'
func SqlParse(str string, values ...any) string {
	var result string
	var valueIndex int
	for i := 0; i < len(str); i++ {
		if str[i] == '?' && valueIndex < len(values) {
			// 找到问号占位符，需要替换
			switch v := values[valueIndex].(type) {
			case string:
				// 字符串类型，添加单引号并转义内部的单引号
				escaped := ""
				for _, ch := range v {
					if ch == '\'' {
						escaped += "''"
					} else {
						escaped += string(ch)
					}
				}
				result += "'" + escaped + "'"
			case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
				// 数字类型，直接转换为字符串
				result += number.String(v)
			case nil:
				// nil值转换为NULL
				result += "NULL"
			default:
				// 其他类型，转换为字符串并添加单引号
				result += "'" + number.String(v) + "'"
			}
			valueIndex++
		} else {
			// 复制原字符
			result += string(str[i])
		}
	}
	return result
}

func MD5(input string) string {
	md5Ctx := md5.New()
	md5Ctx.Write([]byte(input))
	return hex.EncodeToString(md5Ctx.Sum(nil))
}
