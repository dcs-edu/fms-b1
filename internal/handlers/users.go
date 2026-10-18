// Package handlers
package handlers

import (
	// "fmt"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/Jxt-Eli/template/internal/auth"
	"github.com/Jxt-Eli/template/internal/mailer"
	"github.com/Jxt-Eli/template/internal/middleware"
	"github.com/Jxt-Eli/template/internal/models"
	"github.com/Jxt-Eli/template/internal/paystack"
	"github.com/Jxt-Eli/template/internal/repository"
)

type Pool struct {
	Repo      *repository.Repository
	Paystack  *paystack.Client
	Mailer    *mailer.Mailer // sends forgot-password codes
	JWTSecret []byte // signs login tokens; the same secret the JWT middleware checks them with
}

func NewPool(repo *repository.Repository, ps *paystack.Client, m *mailer.Mailer, jwtSecret []byte) *Pool {
	return &Pool{Repo: repo, Paystack: ps, Mailer: m, JWTSecret: jwtSecret}
}

// TEST: TEST HANDLER FUNCTIONS INDEPENDENTLY

// POST /auth/register   {"name", "email", "phone", "password"}
func (srv *Pool) CreateUserHandler(w http.ResponseWriter, r *http.Request) {
	var newUser models.User
	var send models.UserResponse
	ctx := r.Context()

	if err := json.NewDecoder(r.Body).Decode(&newUser); err != nil {
		slog.ErrorContext(ctx, "Json decode error", "error", err)
		http.Error(w, "400 invalid request body", http.StatusBadRequest)
		return
	}
	newUser.Name = strings.TrimSpace(newUser.Name)
	newUser.Email = strings.TrimSpace(newUser.Email)
	newUser.Phone = strings.TrimSpace(newUser.Phone)
	if newUser.Name == "" || newUser.Phone == "" {
		http.Error(w, "400 name and phone are required", http.StatusBadRequest)
		return
	}
	// not a full email check (only sending an email proves an address works), just a guard against typos
	if !strings.Contains(newUser.Email, "@") {
		http.Error(w, "400 a valid email is required", http.StatusBadRequest)
		return
	}
	// same rule as ChangePasswordHandler and ResetPasswordHandler
	if len(newUser.Password) < minPasswordLen {
		http.Error(w, "400 password must be at least 8 characters", http.StatusBadRequest)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newUser.Password), bcrypt.DefaultCost)
	if errors.Is(err, bcrypt.ErrPasswordTooLong) {
		// bcrypt only reads 72 bytes; this was a 500 before, but it's the client's input that's wrong
		http.Error(w, "400 password must be at most 72 bytes", http.StatusBadRequest)
		return
	}
	if err != nil {
		slog.ErrorContext(ctx, "Password hash error", "error", err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}
	newUser.Password = string(hashedPassword)

	createdUser, err := srv.Repo.Create(ctx, &newUser, &send)
	// only a unique violation means "already exists". Every other error used to say that too,
	// so a database outage looked like the user's fault.
	if errors.Is(err, repository.ErrDuplicate) {
		http.Error(w, "409 an account with that email already exists", http.StatusConflict)
		return
	}
	if err != nil {
		slog.ErrorContext(ctx, "Create user error", "error", err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, r, http.StatusCreated, createdUser)
}

// POST /auth/login   {"email", "password"}
// A wrong email and a wrong password get the same answer, so the reply doesn't say which one exists.
func (srv *Pool) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var cred models.Login
	ctx := r.Context()
	if err := json.NewDecoder(r.Body).Decode(&cred); err != nil {
		http.Error(w, "400 invalid request body", http.StatusBadRequest)
		return
	}
	user, err := srv.Repo.GetByEmail(ctx, strings.TrimSpace(cred.Email))
	if errors.Is(err, sql.ErrNoRows) {
		// 401, not 400: the request was well-formed, the credentials just aren't valid
		http.Error(w, "401 invalid email or password", http.StatusUnauthorized)
		return
	}
	if err != nil {
		slog.ErrorContext(ctx, "GetByEmail error", "error", err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(cred.Password)); err != nil {
		http.Error(w, "401 invalid email or password", http.StatusUnauthorized)
		return
	}

	token, err := auth.GenerateToken(srv.JWTSecret, user.ID, user.Email, user.Role)
	if err != nil {
		slog.ErrorContext(ctx, "jwt error", "error", err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, r, http.StatusOK, struct {
		Token   string `json:"token"`
		Message string `json:"message"`
	}{
		Token:   token,
		Message: "login successful",
	})
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

// GET /me
// Who is logged in. Like ChangePasswordHandler, the user comes from the token, never from the request,
// so nobody can ask for someone else's profile.
func (srv *Pool) ProfileHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.UserContextKey).(*auth.CustomClaims)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	ctx := r.Context()

	profile, err := srv.Repo.GetUser(ctx, claims.ID)
	if errors.Is(err, repository.ErrUserNotFound) {
		// valid token, but the account was deleted after it was issued
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if err != nil {
		slog.ErrorContext(ctx, "GetUser error", "error", err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, r, http.StatusOK, profile)
}

