package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Utility is anything a student is billed for each semester (tuition, PTA dues, ...).
type Utility struct {
	ID           int64     `json:"id"          db:"id"`
	UtilName     string    `json:"util_name"   db:"util_name"`
	IsActive     bool      `json:"is_active"   db:"is_active"`
}

// UtilityPrice is the price of one utility for one semester. Every student owes every bill of a semester.
type UtilityPrice struct {
	BillID       int64             `json:"bill_id"     db:"bill_id"`
	UtilID       int64             `json:"util_id"     db:"util_id"`
	Sem          string            `json:"sem"         db:"sem"`
	Amount       decimal.Decimal   `json:"amount"      db:"amount"`
}

// Payment is money a student paid towards a bill. A bill can be settled over several payments.
type Payment struct {
	TxnID        uuid.UUID         `json:"txn_id"       db:"txn_id"`
	BillID       int64             `json:"bill_id"      db:"bill_id"`
	AdmissionNo  uuid.UUID         `json:"admission_no" db:"admission_no"`
	Amount       decimal.Decimal   `json:"amount"       db:"amount"`
	PaidAt       time.Time         `json:"paid_at"      db:"paid_at"`
	Reason       *string           `json:"reason"       db:"reason"`		// nullable: nil <-> NULL
}
