package handlers

import (
	"fmt"
	"encoding/json"
	"net/http"
	"log/slog"


	"github.com/Jxt-Eli/template/internal/middleware"
	"github.com/Jxt-Eli/template/internal/models"
	"github.com/Jxt-Eli/template/internal/auth"
	"github.com/Jxt-Eli/template/pkg"
)

func (srv *Pool) StudentsHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.UserContextKey).(*auth.CustomClaims)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Fuckass logic statement that's just confusing as hell
	if claims.Role != auth.RolePrincipal && claims.Role != auth.RoleAdmin {
		http.Error(w, "forbidden: insufficient privileges", http.StatusForbidden)
		return
	}
	ctx := r.Context()
	var std models.Student

	err := json.NewDecoder(r.Body).Decode(&std);
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		slog.Error("bad request", "error", err)
		return
	}
	std.GradYear, err = pkg.CalculateGradYear(std.Grade)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	newStd, err := srv.Repo.InsertStudent(ctx, std)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		slog.Error("repo error:", "error", err)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(struct{
		StudentInfo models.Student `json:"student_info"`
		StudentID   string         `json:"student_id"`
	}{
		StudentInfo: *newStd,
		StudentID: fmt.Sprintf("%02d-%04d", newStd.GradYear, newStd.SeqNum),
	})
}
