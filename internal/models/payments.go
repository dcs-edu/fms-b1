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

// Payment statuses. Only completed payments are money actually received.
const (
	PaymentPending   = "pending"
	PaymentCompleted = "completed"
	PaymentFailed    = "failed"
)

type Payment struct {
	TxnID        uuid.UUID         `json:"txn_id"                db:"txn_id"`
	BillID       int64             `json:"bill_id"               db:"bill_id"`
	AdmissionNo  uuid.UUID         `json:"admission_no"          db:"admission_no"`
	Amount       decimal.Decimal   `json:"amount"                db:"amount"`
	Status       string            `json:"status"                db:"status"`
	Reference    *string           `json:"reference"             db:"reference"`			// Paystack reference; nil for cash
	CreatedAt    time.Time         `json:"created_at"            db:"created_at"`
	PaidAt       *time.Time        `json:"paid_at"               db:"paid_at"`			// nil until completed
	InitiatedBy  *uuid.UUID        `json:"initiated_by,omitempty" db:"initiated_by"`	// user who started it (staff for cash, parent/bursar online); omitted when nil
}

// NewPayment is the request body for both a staff-recorded payment and a parent starting an online one.
// It takes the short student ID (what's printed on the student's card) instead of the admission number.
type NewPayment struct {
	StudentID    string            `json:"student_id"`
	BillID       int64             `json:"bill_id"`
	Amount       decimal.Decimal   `json:"amount"`			// send as a string ("150.00") so no float rounding happens on the way in
}

// StudentPayments is one student's payment history: who they are, then what they paid.
type StudentPayments struct {
	Student      Std               `json:"student"`
	Payments     []Payment         `json:"payments"`
}
