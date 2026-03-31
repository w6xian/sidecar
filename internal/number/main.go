package number

import (
	"encoding/json"
	"strconv"

	"github.com/govalues/decimal"
)

type NullInt64 struct {
	Int64 int64
	Valid bool
}

func Uint64(value string) uint64 {
	if d, err := strconv.ParseUint(value, 10, 64); err == nil {
		return d
	}
	return 0
}
func Int64(value string) int64 {
	if d, err := strconv.ParseInt(value, 10, 64); err == nil {
		return d
	}
	return 0
}
func Int(value string) int {
	if d, err := strconv.ParseInt(value, 10, 64); err == nil {
		return int(d)
	}
	return 0
}

func JsNumber(value string) float64 {
	if d, err := strconv.ParseFloat(value, 64); err == nil {
		return d
	}
	return 0
}

func Bool(value string) bool {
	if d, err := strconv.ParseBool(value); err == nil {
		return d
	}
	return false
}

func String(value any) string {
	var key string
	if value == nil {
		return key
	}
	switch ft := value.(type) {
	case float64:
		key = strconv.FormatFloat(ft, 'f', -1, 64)
	case float32:
		key = strconv.FormatFloat(float64(ft), 'f', -1, 64)
	case int:
		key = strconv.Itoa(ft)
	case uint:
		key = strconv.Itoa(int(ft))
	case int8:
		key = strconv.Itoa(int(ft))
	case uint8:
		key = strconv.Itoa(int(ft))
	case int16:
		key = strconv.Itoa(int(ft))
	case uint16:
		key = strconv.Itoa(int(ft))
	case int32:
		key = strconv.Itoa(int(ft))
	case uint32:
		key = strconv.Itoa(int(ft))
	case int64:
		key = strconv.FormatInt(ft, 10)
	case uint64:
		key = strconv.FormatUint(ft, 10)
	case string:
		key = value.(string)
	case []byte:
		key = string(value.([]byte))
	default:
		newValue, _ := json.Marshal(value)
		key = string(newValue)
	}
	return key
}

// 两数相加
func Add(a, b float64) float64 {
	ad, _ := decimal.NewFromFloat64(a)
	bd, _ := decimal.NewFromFloat64(b)
	r, _ := ad.Add(bd)
	num, _ := r.Float64()
	return num
}

// 两数相除
func Divide(a, b float64) float64 {
	ad, _ := decimal.NewFromFloat64(a)
	bd, _ := decimal.NewFromFloat64(b)
	r, _ := ad.Quo(bd)
	num, _ := r.Float64()
	return num
}

// 两数相减
func Subtract(a, b float64) float64 {
	ad, _ := decimal.NewFromFloat64(a)
	bd, _ := decimal.NewFromFloat64(b)
	r, _ := ad.Sub(bd)
	num, _ := r.Float64()
	return num
}

// 两数相乘
func Multiply(a, b float64) float64 {
	ad, _ := decimal.NewFromFloat64(a)
	bd, _ := decimal.NewFromFloat64(b)
	r, _ := ad.Mul(bd)
	num, _ := r.Float64()
	return num
}

type commonType interface {
	int | float32 | uint64 | uint8 | int64 | float64
}

// Abs returns the absolute value of a.
func Abs[T commonType](a T) T {
	if a < 0 {
		return -a
	}
	return a
}

// Max returns the larger of a and b.
func Max[T commonType](a, b T) T {
	if a > b {
		return a
	}

	return b
}

// Min returns the smaller of a and b.
func Min[T commonType](a, b T) T {
	if a < b {
		return a
	}
	return b
}

func Round(value float64, precision int) float64 {
	ad, _ := decimal.NewFromFloat64(value)
	r := ad.Round(precision)
	num, _ := r.Float64()
	return num
}

func Round2(value float64) float64 {
	ad, _ := decimal.NewFromFloat64(value)
	r := ad.Round(2)
	num, _ := r.Float64()
	return num
}
