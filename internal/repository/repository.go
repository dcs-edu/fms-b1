// Package repository
package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	// "github.com/Jxt-Eli/template/internal/middleware"
	"github.com/Jxt-Eli/template/internal/models"
)

type Repository struct  {
	DB *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{ DB: db }
}

func (r *Repository) Create (ctx context.Context, user *models.User) (*models.User, error) {
	query :=
	`
		INSERT INTO users (name, phone, email, password_hash) 
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`
	if err := r.DB.GetContext(ctx , user, query, user.Name, /*middleware.RoleTeacher,*/ user.Phone, user.Email, user.Password); err != nil {
		return nil, fmt.Errorf("DATABASE FETCH ERROR %w", err)
	}
	return user, nil
}

func (r *Repository) GetByEmail (ctx context.Context, email string) (*models.User, error) {
	var user models.User
	query := 
	`
		SELECT id, name, role, email, password_hash
		FROM users 
		WHERE email = $1
	`
	if err := r.DB.GetContext(ctx, &user, query, email); err != nil {
		return nil, fmt.Errorf("DATABASE FETCH ERROR %w", err)
	}
	return &user, nil
}
