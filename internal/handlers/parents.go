package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/Jxt-Eli/template/internal/auth"
	"github.com/Jxt-Eli/template/internal/middleware"
	"github.com/Jxt-Eli/template/internal/repository"
	"github.com/Jxt-Eli/template/pkg"
)

// POST /parents/links   {"parent_email": "mum@example.com", "student_id": "30-0042"}
// Staff confirm who a parent is before linking them, so parents can't link themselves to any child.
func (srv *Pool) LinkParentHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.UserContextKey).(*auth.CustomClaims)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if claims.Role != auth.RoleAdmin && claims.Role != auth.RolePrincipal {
		http.Error(w, "forbidden: insufficient privileges", http.StatusForbidden)
		return
	}
	ctx := r.Context()

	var req struct {
		ParentEmail string `json:"parent_email"`
		StudentID   string `json:"student_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "400 invalid request body", http.StatusBadRequest)
		return
	}

	gradYear, seqNum, err := pkg.ParseStudentID(req.StudentID)
	if err != nil {
		http.Error(w, "400 "+err.Error(), http.StatusBadRequest)
		return
	}

	parent, err := srv.Repo.GetByEmail(ctx, req.ParentEmail)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "404 no account with that email", http.StatusNotFound)
		return
	}
	if err != nil {
		slog.ErrorContext(ctx, "GetByEmail error", "error", err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}
	if parent.Role != auth.RoleParent {
		http.Error(w, "400 that account is not a parent account", http.StatusBadRequest)
		return
	}

	std, err := srv.Repo.GetStd(ctx, gradYear, seqNum)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "404 student not found", http.StatusNotFound)
		return
	}
	if err != nil {
		slog.ErrorContext(ctx, "GetStd error", "error", err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}

	err = srv.Repo.LinkParent(ctx, parent.ID, std.AdmissionNo)
	if errors.Is(err, repository.ErrDuplicate) {
		http.Error(w, "409 parent is already linked to this student", http.StatusConflict)
		return
	}
	if err != nil {
		slog.ErrorContext(ctx, "LinkParent error", "error", err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// GET /me/children
func (srv *Pool) MyChildrenHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.UserContextKey).(*auth.CustomClaims)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if claims.Role != auth.RoleParent {
		http.Error(w, "forbidden: only parent accounts have children linked", http.StatusForbidden)
		return
	}
	ctx := r.Context()

	children, err := srv.Repo.ListChildren(ctx, claims.ID)
	if err != nil {
		slog.ErrorContext(ctx, "ListChildren error", "error", err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, r, http.StatusOK, children)
}
