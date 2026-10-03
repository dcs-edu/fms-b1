// Package repository: internal/repository/payments.go
package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/Jxt-Eli/template/internal/models"
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

func (r *Repository) InsertPayment(ctx context.Context, p models.Payment) (*models.Payment, error) {
	if !p.Amount.IsPositive() {
		return nil, ErrNonPositiveAmount
	}

	query :=
	`
		INSERT INTO payments (bill_id, admission_no, amount, reason)
		VALUES ($1, $2, $3, $4)
		RETURNING txn_id, paid_at
	`
	if err := r.DB.QueryRowxContext(ctx, query, p.BillID, p.AdmissionNo, p.Amount, p.Reason).Scan(&p.TxnID, &p.PaidAt); err != nil {
		return nil, fmt.Errorf("insert payment: %w", mapPgError(err))
	}
	return &p, nil
}

// ListPaymentsBetween returns every payment made in the half-open range [from, to):
// a payment at exactly `from` is included, one at exactly `to` is not.
// That way back-to-back ranges (Mon 00:00 -> Tue 00:00, Tue 00:00 -> Wed 00:00) never count a payment twice.
func (r *Repository) ListPaymentsBetween(ctx context.Context, from, to time.Time) ([]models.Payment, error) {
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
