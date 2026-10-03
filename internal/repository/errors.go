// Package repository: internal/repository/errors.go
package repository

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrNonPositiveAmount              = errors.New("amount must be greater than zero")
	ErrDuplicate                      = errors.New("record already exists")
	ErrInvalidReference               = errors.New("referenced record does not exist")
	ErrUndefinedColumn                = errors.New("column does not exist")
	ErrInvalidTransactionInitiation   = errors.New("transaction failed")
	ErrInvalidTimeRange               = errors.New("time range end must be after its start")
)

// Postgres error codes: https://www.postgresql.org/docs/current/errcodes-appendix.html
const (
	pgUniqueViolation              =  "23505"
	pgForeignKeyViolation          =  "23503"
	pgUndefinedColumn              =  "42703"
	pgInvalidTransactionInitiation =  "0B000"
)

func mapPgError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}
	switch pgErr.Code {
	case pgUniqueViolation:
		return fmt.Errorf("%w: %s", ErrDuplicate, pgErr.ConstraintName)
	case pgForeignKeyViolation:
		return fmt.Errorf("%w: %s", ErrInvalidReference, pgErr.ConstraintName)
	case pgUndefinedColumn:
		return fmt.Errorf("%w: %s", ErrUndefinedColumn, pgErr.ConstraintName)
	case pgInvalidTransactionInitiation:
		return fmt.Errorf("%w: %s", ErrInvalidTransactionInitiation, pgErr.ConstraintName)
	}
	return err
}
