// Package handlers
package handlers

import (
	// "fmt"
	"encoding/json"
	"log/slog"
	"net/http"

	"golang.org/x/crypto/bcrypt"

	"github.com/Jxt-Eli/template/internal/auth"
	"github.com/Jxt-Eli/template/internal/models"
	"github.com/Jxt-Eli/template/internal/repository"
)

type Pool struct {
	Repo *repository.Repository
}

func NewPool(repo *repository.Repository) *Pool {
	// if repo == nil {
	// 	panic("cannot initialize repository with nil database connection")
	// }
	return &Pool{Repo: repo}
}

// TEST: TEST HANDLER FUNCTIONS INDEPENDENTLY

func (srv *Pool) CreateUserHandler(w http.ResponseWriter, r *http.Request) {
	var newUser models.User
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
	createdUser, err := srv.Repo.Create(ctx, &newUser)
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

	if err != nil {
		slog.Error("DB FETCH ERROR\n", "error", err)
		http.Error(w, "400 invalid email or password", http.StatusBadRequest)
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), password); err != nil {
		slog.Error("password hash error", "error", err)
		http.Error(w, "400 invalid email or password", http.StatusBadRequest)
		return
	}

	token, err := auth.GenerateToken(user.ID, user.Email, user.Role)
	if err != nil {
		slog.Error("jwt error", "error", err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
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
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
	}
	
}
