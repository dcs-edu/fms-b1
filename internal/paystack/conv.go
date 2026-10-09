package paystack

import (
	// "math"

	"github.com/shopspring/decimal"
)

func ToCoin(amount decimal.Decimal) int64 {
	return amount.Mul(decimal.NewFromInt(100)).IntPart()
}

func ToNote(amount int64) decimal.Decimal {
	a := decimal.NewFromInt(amount)
	return a.Div(decimal.NewFromInt(100))
}
