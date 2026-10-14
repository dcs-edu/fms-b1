package paystack

import (
	// "math"

	"github.com/shopspring/decimal"
)

// Convert Note from decimal.Decimal to coins in int64 form
func ToCoin(amount decimal.Decimal) int64 {
	return amount.Mul(decimal.NewFromInt(100)).IntPart()
}

// Might not be used for reasons of accuracy due to its method of chopping off numbers far after the decimal places
func ToNote(amount int64) decimal.Decimal {
	a := decimal.NewFromInt(amount)
	return a.Div(decimal.NewFromInt(100))
}
