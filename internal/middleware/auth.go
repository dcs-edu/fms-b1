// Package middleware
package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
	"os"
	"log/slog"

	"github.com/Jxt-Eli/template/internal/auth"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
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


func JwtMiddleware(f http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		godotenv.Load()
		jwtSecret := []byte(os.Getenv("JWTSECRET"))
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

		tokenString := parts[1]   // just "eyJhbGc..." — this is what was missing
		claims := &auth.CustomClaims{}
		// INFO: anonymous function ceremony for a fuckton of usecases I'll never hit :( *crying emoji*
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
