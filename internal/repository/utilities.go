package repository


import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/Jxt-Eli/template/internal/models"
)

func (r *Repository) GetUtilities (ctx context.Context) ([]models.Bill, error) {
	query :=
	`
		SELECT up.bill_id, u.util_name, up.sem, up.amount, u.id AS util_id
		FROM utilities u
		JOIN utility_prices up ON u.id = up.util_id
		WHERE u.is_active = true
		ORDER BY up.sem, u.util_name, u.id
	`
	var bills []models.Bill
	if err := r.DB.SelectContext(ctx, &bills, query); err != nil {
		return nil, fmt.Errorf("list utilities: %w", mapPgError(err))
	}
	return bills, nil
}

// StudentBills is every bill with what this student has paid on it and what is left.
// It is remainingOnBill for all bills at once, so the two must agree: only completed payments count.
// WARNING: No is_active filter: switching a utility off must not hide money a student still owes on it.
func (r *Repository) StudentBills(ctx context.Context, admissionNo uuid.UUID) ([]models.StudentBill, error) {
	query :=
	`
		SELECT up.bill_id, u.util_name, up.sem, up.amount,
		       COALESCE(SUM(p.amount), 0)             AS paid,
		       up.amount - COALESCE(SUM(p.amount), 0) AS remaining
		FROM utility_prices up
		JOIN utilities u ON u.id = up.util_id
		LEFT JOIN payments p
		       ON p.bill_id = up.bill_id AND p.admission_no = $1 AND p.status = 'completed'
		GROUP BY up.bill_id, u.util_name
		ORDER BY up.sem, u.util_name
	`
	var bills []models.StudentBill
	if err := r.DB.SelectContext(ctx, &bills, query, admissionNo); err != nil {
		return nil, fmt.Errorf("student bills: %w", mapPgError(err))
	}
	return bills, nil
}

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

func (r *Repository) DisableUtility(ctx context.Context, util int64) error {
	query :=
	`
		UPDATE utilities
		SET is_active = false
		WHERE id = $1
	`
	if _, err := r.DB.ExecContext(ctx, query, util); err != nil {
		return fmt.Errorf("DisableUtility error: %w", mapPgError(err))
	}
	return nil
}
