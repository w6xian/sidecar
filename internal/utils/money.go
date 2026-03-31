package utils

import (
	"math"

	"github.com/Rhymond/go-money"
)

func FormatMoney(amount int64) string {
	// 1/100转成分，只收到分就好
	amount = int64(math.Round(float64(amount) / 100))
	m := money.New(amount, "")
	return m.Display()

}
