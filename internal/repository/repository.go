// Package repository
package repository

import (
	"context"
	"fmt"

	"database/sql"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/Jxt-Eli/template/internal/auth"
	"github.com/Jxt-Eli/template/internal/models"
)

type UserStore interface {
	Create (ctx context.Context, user *models.User) (*models.User, error)
	GetByEmail (ctx context.Context, email string) (*models.User, error)
	UpdateRole (ctx context.Context, user *models.Role, role auth.Role) ( *models.Role, error )
	InsertStudent(ctx context.Context, s models.Student) (*models.Student, error)
}

type Repository struct {
	DB *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	if db == nil {
		panic("cannot initialize repository with nil database connection")
	}
	return &Repository{ DB: db }
}

// TEST: use testcontainers-go
func (r *Repository) Create (ctx context.Context, user *models.User, userResponse *models.UserResponse) (*models.UserResponse, error) {
	query :=
	`
		INSERT INTO users (name, phone, email, password_hash) 
		VALUES ($1, $2, $3, $4)
		RETURNING email, name, role, phone, id, created_at
	`
	if err := r.DB.GetContext(ctx , userResponse, query, user.Name, user.Phone, user.Email, user.Password); err != nil {
		return nil, fmt.Errorf("database fetch error: %w", mapPgError(err))
	}
	return userResponse, nil
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
		return nil, fmt.Errorf("database fetch error: %w", mapPgError(err))
	}
	return &user, nil
}

func (r *Repository) GetPasswordHash (ctx context.Context, id uuid.UUID) (string, error) {
	var hash string
	query :=
	`
		SELECT password_hash
		FROM users
		WHERE id = $1
	`
	if err := r.DB.GetContext(ctx, &hash, query, id); err != nil {
		return "", fmt.Errorf("fetch password hash: %w", mapPgError(err))
	}
	return hash, nil
}

func (r *Repository) UpdatePassword (ctx context.Context, id uuid.UUID, hash string) error {
	query :=
	`
		UPDATE users
		SET password_hash = $1
		WHERE id = $2
	`
	res, err := r.DB.ExecContext(ctx, query, hash, id)
	if err != nil {
		return fmt.Errorf("update password: %w", mapPgError(err))
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("update password: %w", sql.ErrNoRows)
	}
	return nil
}

func (r *Repository) UpdateRole (ctx context.Context, user *models.Role) ( *models.Role, error ) {
	query := 
	`
		UPDATE users 
		SET role = $1
		WHERE email = $2
		RETURNING name, email, role
	`
	if err := r.DB.GetContext(ctx, user, query, user.Role, user.Email); err != nil {
		return nil, fmt.Errorf("UpdateRole error: %w", mapPgError(err))
	}
	return user, nil
}

func (r *Repository) InsertStudent(ctx context.Context, s models.Student) (*models.Student, error) {
	tx, err := r.DB.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction error: %w", mapPgError(err))
	}
	defer tx.Rollback()

	var maxSeq sql.NullInt16
	err = tx.GetContext(ctx, &maxSeq,
`SELECT MAX(seq_num) FROM students WHERE grad_year = $1`, s.GradYear)
	if err != nil {
		return nil, fmt.Errorf("fetch max seq_num: %w", mapPgError(err))
	}

	nextSeq := int16(1)
	if maxSeq.Valid {
		nextSeq = maxSeq.Int16 + 1
	}
	s.SeqNum = nextSeq

	// admission_no and created_at are left out so Postgres fills them (gen_random_uuid(), now()).
	// Sending them from the struct would send Go's zero values: the all-zero UUID and year 1.
	query :=
	`
		INSERT INTO students (fname, mname, lname, grade, dob, gender, nationality, address, guardian, g_contact, g_occupation, e_contact, med_con, allergies, photo, grad_year, seq_num)
		VALUES (:fname, :mname,:lname, :grade, :dob, :gender, :nationality, :address, :guardian, :g_contact, :g_occupation, :e_contact, :med_con, :allergies, :photo, :grad_year, :seq_num)
		RETURNING admission_no, created_at
	`
	stmt, err := tx.PrepareNamedContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("prepare insert student: %w", mapPgError(err))
	}
	defer stmt.Close()

	if err := stmt.GetContext(ctx, &s, s); err != nil {
		return nil, fmt.Errorf("insert student error: %w", mapPgError(err))
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction error: %w", mapPgError(err))
	}

	return &s, nil
}

// func (r *Repository) AddBook (ctx context.Context, subject *models.Book) (error)
