// Package models
package models

import ( 
	"time"
	"github.com/google/uuid"

	"github.com/Jxt-Eli/template/internal/auth"
)

type User struct {
	ID           uuid.UUID `json:"id"         db:"id"`
	Name         string    `json:"name"       db:"name"`
	Role         auth.Role `json:"role"       db:"role"`
	Email        string    `json:"email"      db:"email"`
	Phone        string    `json:"phone"      db:"phone"`
	Password     string    `json:"-"          db:"password_hash"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

type Login struct {
	Name         string    `json:"name"    db:"name"`
	Email        string    `json:"email"   db:"email"`
	Phone        string    `json:"phone"   db:"phone"`
	Password     string    `json:"-"       db:"password_hash"`
}

type Book struct {
	BookName     string    `json:"book_name" db:"book_name"`
	Grade        string    `json:"grade" db:"grade"`
}
