// Package repository
package repository

import (
	"context"
	"fmt"
	"log/slog"

	"database/sql"
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
	// if db == nil {
	// 	panic("cannot initialize repository with nil database connection")
	// }
	return &Repository{ DB: db }
}

// TEST: use testcontainers-go
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

func (r *Repository) UpdateRole (ctx context.Context, user *models.Role, role auth.Role) ( *models.Role, error ) {
	query := 
	`
		UPDATE users 
		SET role = $1
		WHERE email = $2
	`
	if err := r.DB.GetContext(ctx, user, query, user.Email, user.Role); err != nil {
		slog.Error("UpdateRole error: check email and try again", "error", err)
		return nil, err
	}
	return user, nil
}

func (r *Repository) InsertStudent(ctx context.Context, s models.Student) (*models.Student, error) {
	tx, err := r.DB.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction error: %w", err)
	}
	defer tx.Rollback()

	var maxSeq sql.NullInt16
	err = tx.GetContext(ctx, &maxSeq,
		`SELECT MAX(seq_num) FROM students WHERE grad_year = $1`, s.GradYear)
	if err != nil {
		return nil, fmt.Errorf("fetch max seq_num: %w", err)
	}

	nextSeq := int16(1)
	if maxSeq.Valid {
		nextSeq = maxSeq.Int16 + 1
	}
	s.SeqNum = nextSeq

	query := 
	`
		INSERT INTO students (fname, mname, lname, grade, dob, gender, nationality, address, guardian, g_contact, g_occupation, e_contact, med_con, allergies, photo, created_at, admission_no, grad_year, seq_num)
		VALUES (:fname, :mname,:lname, :grade, :dob, :gender, :nationality, :address, :guardian, :g_contact, :g_occupation, :e_contact, :med_con, :allergies, :photo, :created_at, :admission_no, :grad_year, :seq_num)
	`
	if _, err := tx.NamedExecContext(ctx, query, s); err != nil {
		return nil, fmt.Errorf("insert student error: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction error: %w", err)
	}

	return &s, nil
}

// func (r *Repository) AddBook (ctx context.Context, subject *models.Book) (error)
