// Package auth: internal/auth/auth.go
package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Role string

const (
	RoleAdmin       Role   = "admin"
	RolePrincipal   Role   = "principal"
	RoleTeacher     Role   = "teacher"
	RoleBursar      Role   = "bursar"
	RoleParent      Role   = "parent"
)

type CustomClaims struct {
	ID uuid.UUID `json:"id"`
	Email string `json:"email"`
	Role  Role `json:"role"`
	jwt.RegisteredClaims
}

// GenerateToken takes the secret as an argument instead of reading .env itself:
// main reads it once at startup and passes it down.
func GenerateToken(jwtSecret []byte, id uuid.UUID, email string, role Role) (string, error) {
	claims := CustomClaims {
		ID: id,
		Email: email,
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24*time.Hour)),
			IssuedAt: jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}
