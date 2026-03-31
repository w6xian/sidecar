package utils

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"unicode"

	"github.com/mozillazg/go-pinyin"
)

var others = map[string]string{}

func Pinyin(s string) string {
	a := pinyin.NewArgs()
	// a.Style = pinyin.Initials
	rst := []string{}
	for _, r := range s {
		if unicode.IsSpace(r) {
			continue
		}
		if unicode.IsUpper(r) || unicode.IsLower(r) || unicode.IsNumber(r) {
			rst = append(rst, strings.ToLower(string(r)))
			continue
		}
		if unicode.IsNumber(r) {
			rst = append(rst, string(r))
			continue
		}
		py := pinyin.LazyPinyin(string(r), a)
		if len(py) > 0 {
			if len(py[0]) > 0 {
				rst = append(rst, string(py[0][0]))
				continue
			}
			if v, ok := others[string(r)]; ok {
				rst = append(rst, v)
			}
		}
	}
	return strings.Join(rst, "")
}

func MapKeys[T comparable](v map[T]any) []T {
	var keys []T
	for k := range v {
		keys = append(keys, k)
	}
	return keys
}

func JsonString(v any) string {
	t := reflect.TypeOf(v)
	b, err := json.Marshal(v)
	if err != nil {
		switch t.Kind() {
		case reflect.Slice:
			return "[]"
		case reflect.Map:
			return "{}"
		case reflect.Bool:
			return strconv.FormatBool(v.(bool))
		case reflect.String:
			return v.(string)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return strconv.FormatInt(v.(int64), 10)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return strconv.FormatUint(v.(uint64), 10)
		case reflect.Float32, reflect.Float64:
			return strconv.FormatFloat(v.(float64), 'f', -1, 64)
		default:
			return ""
		}

	}
	return string(b)
}
