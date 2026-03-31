package utils

import (
	"github.com/w6xian/sidecar/internal/number"
)

// 格式化数量，2为散称，1为预包装
// 散称 约定放大1000，
func FormatNum(num, style int64) float64 {
	if style == 2 {
		return number.Divide(float64(num), float64(1000))
	}
	return float64(num)
}
func ParseNum(num float64, style int64) int64 {
	if style == 2 {
		return int64(number.Multiply(num, float64(1000)))
	}
	return int64(num)
}

func PackBox(num int64, style int64, pkAmount int64) (float64, int64) {
	pkAmount = max(pkAmount, 1)
	if num == 0 {
		return 0, 0
	}
	if style != 1 {
		numf := FormatNum(num, style)
		pack := number.Divide(numf, float64(pkAmount))
		return pack, 0
	}
	pack := num / pkAmount
	box := num % pkAmount
	return float64(pack), box
}

func CalcPrice(num int64, style int64, pkAmount int64, unitPrice, packPrice int64) float64 {
	pack, box := PackBox(num, style, pkAmount)
	pp := number.Multiply(pack, float64(packPrice))
	up := number.Multiply(float64(box), float64(unitPrice))
	return number.Add(pp, up)
}
