// Package auth: internal/auth/auth.go
package auth

import (
	"time"
	"os"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

type Role string

const (
	RoleAdmin       Role   = "admin"
	RolePrincipal   Role   = "principal"
	RoleTeacher     Role   = "teacher"
)

type CustomClaims struct {
	ID uuid.UUID `json:"id"`
	Email string `json:"email"`
	Role  Role `json:"role"`
	jwt.RegisteredClaims
}

func GenerateToken(id uuid.UUID, email string, role Role) (string, error) {

	godotenv.Load()
	jwtSecret := os.Getenv("JWTSECRET")
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
	return token.SignedString([]byte(jwtSecret))
}
