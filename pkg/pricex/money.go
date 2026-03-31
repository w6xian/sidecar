package pricex

import (
	"github.com/w6xian/sidecar/internal/number"
)

type Money struct {
	value    float64
	Currency string
}

func (m *Money) StoreValue() int64 {
	return int64(number.Multiply(m.value, 10000))
}

func (m *Money) CalcValue() float64 {
	return m.value
}
func (m *Money) InputValue() float64 {
	return m.value
}
func NewMoneyFromStore(value int64) *Money {
	return &Money{
		value:    number.Divide(float64(value), 10000),
		Currency: "CNY",
	}
}
func NewMoneyFromInput(value float64) *Money {
	return &Money{
		value:    value,
		Currency: "CNY",
	}
}
