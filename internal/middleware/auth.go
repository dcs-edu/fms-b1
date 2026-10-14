// Package middleware
package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
	"log/slog"

	"github.com/Jxt-Eli/template/internal/auth"
	"github.com/golang-jwt/jwt/v5"
)

type ctxKey string
const UserContextKey  ctxKey = "claims"


func LoggingMiddleware(f http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
		fmt.Println(r.URL.Path)
		start := time.Now()
		f.ServeHTTP(w, r)
		end := time.Since(start)
		fmt.Printf("t = %v", end)
	})
}

// Config holds what JwtMiddleware needs. main builds it once at startup, so the secret is read once,
// not on every request. Its method is the middleware: cfg.JwtMiddleware has the secret in reach.
type Config struct {
	JWTSecret []byte
}

func (cfg Config) JwtMiddleware(f http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jwtSecret := cfg.JWTSecret
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "missing authorization header", http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "invalid authorization header format", http.StatusUnauthorized)
			return
		}

		tokenString := parts[1]   // "eyJhbGc..."
		claims := &auth.CustomClaims{}
		// anonymous function boilerplate ceremony for a fuckton of usecases I'll never hit :(
		token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) { return jwtSecret, nil })
		if err != nil || !token.Valid{
			slog.Error("Invalid or failed jwt", "error", err)
			http.Error(w, "invalid or expired token", http.StatusUnauthorized)
			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, UserContextKey, claims)
		r = r.WithContext(ctx)

		f.ServeHTTP(w, r)
	})
}
