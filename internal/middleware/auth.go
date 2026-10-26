// Package middleware
package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"
	"log/slog"

	"github.com/Jxt-Eli/template/internal/auth"
	"github.com/golang-jwt/jwt/v5"
)

type ctxKey string
const UserContextKey  ctxKey = "claims"


// statusRecorder remembers the status code a handler sent, so the logger can print it afterwards.
// It embeds http.ResponseWriter, so every method it doesn't define (Header, Write) goes straight to the
// real writer; only WriteHeader is intercepted. A handler that never calls WriteHeader sent a 200.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (rec *statusRecorder) WriteHeader(code int) {
	rec.status = code
	rec.ResponseWriter.WriteHeader(code)
}

func LoggingMiddleware(f http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		f.ServeHTTP(rec, r)
		end := time.Since(start)
		slog.Info("http request handled",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration", end.String(),
			// behind nginx, RemoteAddr is always nginx itself; nginx puts the client's address in X-Real-IP
			// (proxy_set_header X-Real-IP $remote_addr;). Anyone can send that header when there's no nginx
			// in front, so it's only logged, never trusted for decisions.
			"remote", r.RemoteAddr,
			"real_ip", r.Header.Get("X-Real-IP"),
			)
	})
}

// TODO: Remove and implement with nginx

// MaxBodyMiddleware caps every request body at 1 MB. Without it, a stranger could send /auth/register a
// 5 GB body and the JSON decoder would keep reading. Past the limit, reads fail and the handler's
// decode error answers 400. nginx's client_max_body_size does the same job one step earlier.
func MaxBodyMiddleware(f http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		f.ServeHTTP(w, r)
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
		// WithValidMethods pins the algorithm to the one GenerateToken signs with. Without it the token's own
		// header picks the algorithm, and "let the attacker choose how to check their token" is a classic JWT hole.
		token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) { return jwtSecret, nil },
			jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
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
