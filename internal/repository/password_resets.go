// Package repository: internal/repository/password_resets.go
package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

// SavePasswordReset stores a fresh reset code for a user, replacing any older one, and starts its
// attempt count at zero. It reports false when the user's last code is less than a minute old: then
// nothing is saved and no email should be sent, which stops someone flooding a parent's inbox.
//
// ON CONFLICT (user_id) DO UPDATE is an "upsert": insert a row, or update the existing one if the user
// already has a row. Its WHERE decides whether the update happens at all.
func (r *Repository) SavePasswordReset(ctx context.Context, userID uuid.UUID, codeHash string) (bool, error) {
	query :=
	`
		INSERT INTO password_resets (user_id, code_hash, expires_at)
		VALUES ($1, $2, now() + interval '15 minutes')
		ON CONFLICT (user_id) DO UPDATE
		SET code_hash = EXCLUDED.code_hash, expires_at = EXCLUDED.expires_at, attempts = 0, created_at = now()
		WHERE password_resets.created_at < now() - interval '1 minute'
	`
	res, err := r.DB.ExecContext(ctx, query, userID, codeHash)
	if err != nil {
		return false, fmt.Errorf("save password reset: %w", mapPgError(err))
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("save password reset: %w", err)
	}
	return n == 1, nil
}

// UseResetAttempt spends one of the user's 5 guesses and returns the stored code hash to compare against.
// It returns sql.ErrNoRows when there is no code, it has expired, or the guesses are used up.
//
// Checking "attempts < 5" and adding 1 happen in the same UPDATE, so two guesses sent at the same moment
// can't both read "4 so far" and both get through: Postgres runs them one after the other.
// It's the same idea as CompletePayment's "AND status <> 'completed'".
func (r *Repository) UseResetAttempt(ctx context.Context, userID uuid.UUID) (string, error) {
	var codeHash string
	query :=
	`
		UPDATE password_resets
		SET attempts = attempts + 1
		WHERE user_id = $1 AND expires_at > now() AND attempts < 5
		RETURNING code_hash
	`
	if err := r.DB.GetContext(ctx, &codeHash, query, userID); err != nil {
		return "", fmt.Errorf("use reset attempt: %w", mapPgError(err))
	}
	return codeHash, nil
}

// ResetPassword sets the new password and deletes the code, in one transaction, so a code works once.
// Deleting first, by both user and hash, means that if two requests race with the right code, only the
// one whose DELETE finds the row goes on to change the password. The other gets sql.ErrNoRows.
func (r *Repository) ResetPassword(ctx context.Context, userID uuid.UUID, codeHash, passwordHash string) error {
	tx, err := r.DB.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", mapPgError(err))
	}
	defer tx.Rollback() // no-op once Commit succeeds

	res, err := tx.ExecContext(ctx, `DELETE FROM password_resets WHERE user_id = $1 AND code_hash = $2`, userID, codeHash)
	if err != nil {
		return fmt.Errorf("delete password reset: %w", mapPgError(err))
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete password reset: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("delete password reset: %w", sql.ErrNoRows)
	}

	if _, err := tx.ExecContext(ctx, `UPDATE users SET password_hash = $1 WHERE id = $2`, passwordHash, userID); err != nil {
		return fmt.Errorf("reset password: %w", mapPgError(err))
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit password reset: %w", mapPgError(err))
	}
	return nil
}
