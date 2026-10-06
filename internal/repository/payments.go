// Package repository: internal/repository/payments.go
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/Jxt-Eli/template/internal/models"
	"github.com/Jxt-Eli/template/pkg"
)

func (r *Repository) InsertUtility(ctx context.Context, u models.Utility) (*models.Utility, error) {
	query :=
	`
		INSERT INTO utilities (util_name, is_active)
		VALUES ($1, $2)
		RETURNING id
	`
	if err := r.DB.GetContext(ctx, &u.ID, query, u.UtilName, u.IsActive); err != nil {
		return nil, fmt.Errorf("insert utility: %w", mapPgError(err))
	}
	return &u, nil
}

func (r *Repository) InsertUtilityPrice(ctx context.Context, p models.UtilityPrice) (*models.UtilityPrice, error) {
	if !p.Amount.IsPositive() {
		return nil, ErrNonPositiveAmount
	}

	query :=
	`
		INSERT INTO utility_prices (util_id, sem, amount)
		VALUES ($1, $2, $3)
		RETURNING bill_id
	`
	if err := r.DB.GetContext(ctx, &p.BillID, query, p.UtilID, p.Sem, p.Amount); err != nil {
		return nil, fmt.Errorf("insert utility price: %w", mapPgError(err))
	}
	return &p, nil
}

// InsertPayment records a payment towards a bill, refusing anything that would pay the bill past its price.
//
// It reads (how much is already paid) and then writes (the new payment), so it runs in a transaction and
// locks the student's row first. Two payments for the same student arriving together then run one after
// the other: the second one waits, sees the first one's amount in the sum, and can't overpay.
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

	var price decimal.Decimal
	err = tx.GetContext(ctx, &price, `SELECT amount FROM utility_prices WHERE bill_id = $1`, p.BillID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrBillNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("fetch bill price: %w", mapPgError(err))
	}

	var paid decimal.Decimal
	err = tx.GetContext(ctx, &paid,
		`SELECT COALESCE(SUM(amount), 0) FROM payments WHERE bill_id = $1 AND admission_no = $2`,
		p.BillID, p.AdmissionNo)
	if err != nil {
		return nil, fmt.Errorf("sum earlier payments: %w", mapPgError(err))
	}

	if remaining := price.Sub(paid); p.Amount.GreaterThan(remaining) {
		return nil, fmt.Errorf("%w: %s left to pay", ErrOverpayment, remaining.StringFixed(2))
	}

	query :=
	`
		INSERT INTO payments (bill_id, admission_no, amount, reason)
		VALUES ($1, $2, $3, $4)
		RETURNING txn_id, paid_at
	`
	if err := tx.QueryRowxContext(ctx, query, p.BillID, p.AdmissionNo, p.Amount, p.Reason).Scan(&p.TxnID, &p.PaidAt); err != nil {
		return nil, fmt.Errorf("insert payment: %w", mapPgError(err))
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit payment: %w", mapPgError(err))
	}
	return &p, nil
}

// ListPaymentsInRange returns every payment made in the half-open range [from, to):
// a payment at exactly `from` is included, one at exactly `to` is not.
// That way back-to-back ranges (Mon 00:00 -> Tue 00:00, Tue 00:00 -> Wed 00:00) never count a payment twice.
func (r *Repository) ListPaymentsInRange(ctx context.Context, from, to time.Time) ([]models.Payment, error) {
	if !to.After(from) {
		return nil, ErrInvalidTimeRange
	}

	query :=
	`
		SELECT txn_id, bill_id, admission_no, amount, paid_at, reason
		FROM payments
		WHERE paid_at >= $1 AND paid_at < $2
		ORDER BY paid_at
	`
	// non-nil so an empty range encodes as [] in JSON, not null
	payments := []models.Payment{}
	if err := r.DB.SelectContext(ctx, &payments, query, from, to); err != nil {
		return nil, fmt.Errorf("list payments: %w", mapPgError(err))
	}
	return payments, nil
}

// GetStd finds a student by the two columns their student ID is built from.
// It also loads admission_no, so callers can go on to query the tables that reference it.
func (r *Repository) GetStd(ctx context.Context, gradYear, seqNum int16) (*models.Std, error) {
	var std models.Std
	query :=
	`
		SELECT admission_no, fname, lname, grade
		FROM students
		WHERE grad_year = $1 AND seq_num = $2
	`
	if err := r.DB.GetContext(ctx, &std, query, gradYear, seqNum); err != nil {
		return nil, fmt.Errorf("get student: %w", mapPgError(err))
	}
	std.StudentID = pkg.FormatStudentID(gradYear, seqNum)
	return &std, nil
}

// GetIndividualPaymentsInRange is ListPaymentsInRange narrowed to one student.
func (r *Repository) GetIndividualPaymentsInRange(ctx context.Context, from, to time.Time, admissionNo uuid.UUID) ([]models.Payment, error) {
	if !to.After(from) {
		return nil, ErrInvalidTimeRange
	}

	query :=
	`
		SELECT txn_id, bill_id, admission_no, amount, paid_at, reason
		FROM payments
		WHERE admission_no = $1 AND paid_at >= $2 AND paid_at < $3
		ORDER BY paid_at
	`
	// non-nil so an empty range encodes as [] in JSON, not null
	individualPayments := []models.Payment{}
	if err := r.DB.SelectContext(ctx, &individualPayments, query, admissionNo, from, to); err != nil {
		return nil, fmt.Errorf("list student payments: %w", mapPgError(err))
	}
	return individualPayments, nil
}
