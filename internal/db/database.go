// Package db
package db

import (
	"fmt"
	"time"
	"context"
	// "log/slog"

	_"github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

func Connect(dsn string) (*sqlx.DB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3 * time.Second)
	defer cancel()

	db, err := sqlx.ConnectContext(ctx, "pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("\nconnection failed:\n, %w", err)
	}
	
	// TODO: move this stuff to a toml config file (if necessary)
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(3)
	db.SetConnMaxLifetime(12 * time.Minute)
	db.SetConnMaxIdleTime(1 * time.Minute)
	
	return db, nil
}
