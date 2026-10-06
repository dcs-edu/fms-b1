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

type UtilityPrice struct {
	BillID       int64             `json:"bill_id"     db:"bill_id"`
	UtilID       int64             `json:"util_id"     db:"util_id"`
	Sem          string            `json:"sem"         db:"sem"`
	Amount       decimal.Decimal   `json:"amount"      db:"amount"`
}

type Payment struct {
	TxnID        uuid.UUID         `json:"txn_id"       db:"txn_id"`
	BillID       int64             `json:"bill_id"      db:"bill_id"`
	AdmissionNo  uuid.UUID         `json:"admission_no" db:"admission_no"`
	Amount       decimal.Decimal   `json:"amount"       db:"amount"`
	PaidAt       time.Time         `json:"paid_at"      db:"paid_at"`
	Reason       *string           `json:"reason"       db:"reason"`		// nullable: nil <-> NULL
}

// NewPayment is what staff send to record a payment. It takes the short student ID
// (what's printed on the student's card) instead of the admission number.
type NewPayment struct {
	StudentID    string            `json:"student_id"`
	BillID       int64             `json:"bill_id"`
	Amount       decimal.Decimal   `json:"amount"`			// send as a string ("150.00") so no float rounding happens on the way in
	Reason       *string           `json:"reason"`			// e.g. "cash, receipt #0042"
}

// StudentPayments is one student's payment history: who they are, then what they paid.
type StudentPayments struct {
	Student      Std               `json:"student"`
	Payments     []Payment         `json:"payments"`
}
