package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// sets of books are sold as a package for each academic year
type BookPacks struct {
	PackID     string              `json:"book_name"   db:"book_name"`
	Amount       int               `json:"amount"      db:"amount"`
	StockCount   string            `json:"stock_count" db:"stock_count"`
	Grade        string            `json:"grade"       db:"grade"`
	Price        decimal.Decimal   `json:"price"       db:"price"`
	AddedAt      time.Time         `json:"added_at"    db:"added_at"`
}
