// Package repository: internal/repository/payments.go
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"

	"github.com/Jxt-Eli/template/internal/models"
)

// remainingOnBill is what a student still owes on a bill: its price minus their completed payments.
//
// It takes a sqlx.QueryerContext instead of using r.DB, so the same query can run inside a transaction
// (InsertPayment passes its tx) or on its own (RemainingOnBill passes r.DB). Both types satisfy that interface.
func remainingOnBill(ctx context.Context, q sqlx.QueryerContext, admissionNo uuid.UUID, billID int64) (decimal.Decimal, error) {
	var price decimal.Decimal
	err := sqlx.GetContext(ctx, q, &price, `SELECT amount FROM utility_prices WHERE bill_id = $1`, billID)
	if errors.Is(err, sql.ErrNoRows) {
		return decimal.Zero, ErrBillNotFound
	}
	if err != nil {
		return decimal.Zero, fmt.Errorf("fetch bill price: %w", mapPgError(err))
	}

	var paid decimal.Decimal
	query :=
	`
		SELECT COALESCE(SUM(amount), 0)
		FROM payments
		WHERE bill_id = $1 AND admission_no = $2 AND status = 'completed'
	`
	if err := sqlx.GetContext(ctx, q, &paid, query, billID, admissionNo); err != nil {
		return decimal.Zero, fmt.Errorf("sum completed payments: %w", mapPgError(err))
	}
	return price.Sub(paid), nil
}

// RemainingOnBill is the check run before sending a parent to Paystack.
// It can't stop two attempts racing each other: two pending payments can both pass it, and if both are
// paid, the webhook completes both (money that already arrived can't be refused). The bill then shows a
// negative remaining, and staff refund the difference.
func (r *Repository) RemainingOnBill(ctx context.Context, admissionNo uuid.UUID, billID int64) (decimal.Decimal, error) {
	return remainingOnBill(ctx, r.DB, admissionNo, billID)
}

// InsertPayment records a cash payment taken by an admin or principal. It is completed the moment it's saved.
//
// It reads (how much is still owed) and then writes (the new payment), so it runs in a transaction and
// locks the student's row first. Two payments for the same student arriving together then run one after
// the other: the second one waits, sees the first one in the sum, and can't overpay.
func (r *Repository) InsertPayment(ctx context.Context, p models.Payment) (*models.Payment, error) {
	if !p.Amount.IsPositive() {
		return nil, ErrNonPositiveAmount
	}

	tx, err := r.DB.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", mapPgError(err))
	}
	defer tx.Rollback() // no-op once Commit succeeds

	var locked int
	err = tx.GetContext(ctx, &locked, `SELECT 1 FROM students WHERE admission_no = $1 FOR UPDATE`, p.AdmissionNo)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrStudentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lock student: %w", mapPgError(err))
	}

	remaining, err := remainingOnBill(ctx, tx, p.AdmissionNo, p.BillID)
	if err != nil {
		return nil, err
	}
	if p.Amount.GreaterThan(remaining) {
		return nil, fmt.Errorf("%w: %s left to pay", ErrOverpayment, remaining.StringFixed(2))
	}

	query :=
	`
		INSERT INTO payments (bill_id, admission_no, amount, status, paid_at, initiated_by)
		VALUES ($1, $2, $3, 'completed', now(), $4)
		RETURNING txn_id, status, created_at, paid_at
	`
	if err := tx.QueryRowxContext(ctx, query, p.BillID, p.AdmissionNo, p.Amount, p.InitiatedBy).
		Scan(&p.TxnID, &p.Status, &p.CreatedAt, &p.PaidAt); err != nil {
		return nil, fmt.Errorf("insert payment: %w", mapPgError(err))
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit payment: %w", mapPgError(err))
	}
	return &p, nil
}

// InsertPendingPayment saves an online payment attempt before the parent is sent to Paystack.
// The row remembers the student, bill and amount, so the webhook only needs the reference to find it.
func (r *Repository) InsertPendingPayment(ctx context.Context, p models.Payment) (*models.Payment, error) {
	if !p.Amount.IsPositive() {
		return nil, ErrNonPositiveAmount
	}

	query :=
	`
		INSERT INTO payments (bill_id, admission_no, amount, reference, initiated_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING txn_id, status, created_at
	`
	if err := r.DB.QueryRowxContext(ctx, query, p.BillID, p.AdmissionNo, p.Amount, p.Reference, p.InitiatedBy).
		Scan(&p.TxnID, &p.Status, &p.CreatedAt); err != nil {
		return nil, fmt.Errorf("insert pending payment: %w", mapPgError(err))
	}
	return &p, nil
}

func (r *Repository) GetPaymentByReference(ctx context.Context, reference string) (*models.Payment, error) {
	var p models.Payment
	query :=
	`
		SELECT txn_id, bill_id, admission_no, amount, status, reference, created_at, paid_at, initiated_by
		FROM payments
		WHERE reference = $1
	`
	if err := r.DB.GetContext(ctx, &p, query, reference); err != nil {
		return nil, fmt.Errorf("get payment by reference: %w", mapPgError(err))
	}
	return &p, nil
}

// CompletePayment marks a payment as paid. It reports false when nothing changed: the reference is
// unknown, or the payment was already completed. That makes a repeated webhook harmless.
//
// A failed payment can still become completed: a parent whose card was declined can retry on the same
// Paystack page, and if that retry succeeds the money is real. Money received always wins.
func (r *Repository) CompletePayment(ctx context.Context, reference string) (bool, error) {
	query :=
	`
		UPDATE payments
		SET status = 'completed', paid_at = now()
		WHERE reference = $1 AND status <> 'completed'
	`
	res, err := r.DB.ExecContext(ctx, query, reference)
	if err != nil {
		return false, fmt.Errorf("complete payment: %w", mapPgError(err))
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("complete payment: %w", err)
	}
	return n == 1, nil
}

// FailPayment marks a pending payment as failed. Same "false = nothing changed" rule as CompletePayment.
func (r *Repository) FailPayment(ctx context.Context, reference string) (bool, error) {
	query :=
	`
		UPDATE payments
		SET status = 'failed'
		WHERE reference = $1 AND status = 'pending'
	`
	res, err := r.DB.ExecContext(ctx, query, reference)
	if err != nil {
		return false, fmt.Errorf("fail payment: %w", mapPgError(err))
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("fail payment: %w", err)
	}
	return n == 1, nil
}

// ListPaymentsInRange returns every payment attempt started in the half-open range [from, to):
// one at exactly `from` is included, one at exactly `to` is not, so back-to-back ranges never overlap.
// It returns all statuses; callers that add up money must keep only completed ones.
func (r *Repository) ListPaymentsInRange(ctx context.Context, from, to time.Time) ([]models.Payment, error) {
	if !to.After(from) {
		return nil, ErrInvalidTimeRange
	}

	query :=
	`
		SELECT p.txn_id, p.bill_id, p.admission_no, p.amount, p.status, p.reference, p.created_at, p.paid_at, p.initiated_by, s.seq_num, s.grad_year, s.fname, s.mname, s.lname
		FROM payments p
		JOIN students s on p.admission_no = s.admission_no
		WHERE p.created_at >= $1 AND p.created_at < $2
		ORDER BY p.created_at
	`
	// non-nil so an empty range encodes as [] in JSON, not null
	payments := []models.Payment{}
	if err := r.DB.SelectContext(ctx, &payments, query, from, to); err != nil {
		return nil, fmt.Errorf("list payments error: %w", mapPgError(err))
	}
	return payments, nil
}

// GetIndividualPaymentsInRange is ListPaymentsInRange narrowed to one student. (not exactly the same method tho)
func (r *Repository) GetIndividualPaymentsInRange(ctx context.Context, from, to time.Time, admissionNo uuid.UUID) ([]models.Payment, error) {
	if !to.After(from) {
		return nil, ErrInvalidTimeRange
	}

	query :=
	`
		SELECT p.txn_id, p.bill_id, p.admission_no, p.amount, p.status, p.reference, p.created_at, p.paid_at, p.initiated_by, s.seq_num, s.grad_year
		FROM payments p
		JOIN students s ON s.admission_no = p.admission_no
		WHERE p.admission_no = $1 AND p.created_at >= $2 AND p.created_at < $3
		ORDER BY p.created_at
	`
	individualPayments := []models.Payment{}
	if err := r.DB.SelectContext(ctx, &individualPayments, query, admissionNo, from, to); err != nil {
		return nil, fmt.Errorf("list student payments: %w", mapPgError(err))
	}
	return individualPayments, nil
}
