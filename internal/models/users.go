package models

import (
	"time"

	"github.com/google/uuid"

	"github.com/Jxt-Eli/template/internal/auth"
)

type User struct {
	ID           uuid.UUID `json:"id"          db:"id"`
	Name         string    `json:"name"        db:"name"`
	Role         auth.Role `json:"role"        db:"role"`
	Email        string    `json:"email"       db:"email"`
	Phone        string    `json:"phone"       db:"phone"`
	Password     string    `json:"password"    db:"password_hash"`
	CreatedAt    time.Time `json:"created_at"  db:"created_at"`
}

type UserResponse struct {
	ID           uuid.UUID `json:"id"          db:"id"`
	Name         string    `json:"name"        db:"name"`
	Role         auth.Role `json:"role"        db:"role"`
	Email        string    `json:"email"       db:"email"`
	Phone        string    `json:"phone"       db:"phone"`
	Password     string    `json:"-"           db:"password_hash"`
	CreatedAt    time.Time `json:"created_at"  db:"created_at"`
}

type Login struct {
	Name         string    `json:"name"        db:"name"`
	Email        string    `json:"email"       db:"email"`
	Phone        string    `json:"phone"       db:"phone"`
	Password     string    `json:"password"    db:"password_hash"`
}

// ChangePassword is the request body for a logged-in user changing their own password.
type ChangePassword struct {
	OldPassword  string    `json:"old_password"`
	NewPassword  string    `json:"new_password"`
}

type Role struct {
	Name         string    `json:"name"        db:"name"`
	Email        string    `json:"email"       db:"email"`
	Role         auth.Role `json:"role"        db:"role"`
}
