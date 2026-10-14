// Package handlers
package handlers

import (
	// "fmt"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"golang.org/x/crypto/bcrypt"

	"github.com/Jxt-Eli/template/internal/auth"
	"github.com/Jxt-Eli/template/internal/middleware"
	"github.com/Jxt-Eli/template/internal/models"
	"github.com/Jxt-Eli/template/internal/paystack"
	"github.com/Jxt-Eli/template/internal/repository"
)

type Pool struct {
	Repo      *repository.Repository
	Paystack  *paystack.Client
	JWTSecret []byte // signs login tokens; the same secret the JWT middleware checks them with
}

func NewPool(repo *repository.Repository, ps *paystack.Client, jwtSecret []byte) *Pool {
	return &Pool{Repo: repo, Paystack: ps, JWTSecret: jwtSecret}
}

// TEST: TEST HANDLER FUNCTIONS INDEPENDENTLY

func (srv *Pool) CreateUserHandler(w http.ResponseWriter, r *http.Request) {
	var newUser models.User
	var send models.UserResponse
	ctx := r.Context()

	if err := json.NewDecoder(r.Body).Decode(&newUser); err != nil {
		slog.ErrorContext(ctx, "Json decode error", "error", err)
		http.Error(w, "400 invalid request body", http.StatusBadRequest)
		return
	}
	plainTextPassword := []byte(newUser.Password)
	hashedPassword, err := bcrypt.GenerateFromPassword(plainTextPassword, bcrypt.DefaultCost)
	if err != nil {
		slog.ErrorContext(ctx, "Password hash error:", "error", err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}
	newUser.Password = string(hashedPassword)
	createdUser, err := srv.Repo.Create(ctx, &newUser, &send)
	if err != nil {
		http.Error(w, "user already exists", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(createdUser); err != nil {
		slog.ErrorContext(ctx, "Json Encode error: ", "error", err)
		return
	}
}

func (srv *Pool) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var cred models.Login
	ctx := r.Context()
	if err := json.NewDecoder(r.Body).Decode(&cred); err != nil {
		http.Error(w, "400 invalid request body", http.StatusBadRequest)
		return
	}
	password := []byte(cred.Password)
	user, err := srv.Repo.GetByEmail(ctx, cred.Email)
	if errors.Is(err, sql.ErrNoRows) || err != nil {
		slog.ErrorContext(ctx, "db fetch error: %w", "error", err)
		http.Error(w, "400 invalid email or password", http.StatusBadRequest)
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), password); err != nil {
		slog.Error("password hash error", "error", err)
		http.Error(w, "400 invalid email or password", http.StatusBadRequest)
		return
	}

	token, err := auth.GenerateToken(srv.JWTSecret, user.ID, user.Email, user.Role)
	if err != nil {
		slog.Error("jwt error", "error", err)
		http.Error(w, "400 invalid email or password", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(struct{
		Token     string `json:"token"`
		Message   string `json:"message"`
	}{
			Token: token,
			Message: "login successful",
		})
	if err != nil {
		slog.Error("json encode error", "error", err)
		http.Error(w, "400 invalid email or password", http.StatusBadRequest)
	}
	
}

const minPasswordLen = 8

// ChangePasswordHandler lets a logged-in user replace their own password.
// The user comes from the JWT claims, never from the request body, so nobody can change someone else's password.
func (srv *Pool) ChangePasswordHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.UserContextKey).(*auth.CustomClaims)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	ctx := r.Context()

	var req models.ChangePassword
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "400 invalid request body", http.StatusBadRequest)
		return
	}
	if len(req.NewPassword) < minPasswordLen {
		http.Error(w, "400 new password must be at least 8 characters", http.StatusBadRequest)
		return
	}
	if req.NewPassword == req.OldPassword {
		http.Error(w, "400 new password must differ from the old one", http.StatusBadRequest)
		return
	}

	currentHash, err := srv.Repo.GetPasswordHash(ctx, claims.ID)
	if errors.Is(err, sql.ErrNoRows) {
		// valid token, but the account was deleted after it was issued
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if err != nil {
		slog.ErrorContext(ctx, "fetch password hash", "error", err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(req.OldPassword)); err != nil {
		http.Error(w, "400 old password is incorrect", http.StatusBadRequest)
		return
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if errors.Is(err, bcrypt.ErrPasswordTooLong) {
		http.Error(w, "400 new password must be at most 72 bytes", http.StatusBadRequest)
		return
	}
	if err != nil {
		slog.ErrorContext(ctx, "password hash error", "error", err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err := srv.Repo.UpdatePassword(ctx, claims.ID, string(newHash)); err != nil {
		slog.ErrorContext(ctx, "update password", "error", err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
