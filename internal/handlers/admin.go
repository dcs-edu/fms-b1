// Package handlers : Admin endpoint
package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"slices"

	"github.com/Jxt-Eli/template/internal/auth"
	"github.com/Jxt-Eli/template/internal/middleware"
	"github.com/Jxt-Eli/template/internal/models"
)

var knownRoles = []auth.Role{auth.RoleAdmin, auth.RolePrincipal, auth.RoleTeacher, auth.RoleBursar, auth.RoleParent}

// PATCH /admin/users/role   {"email": "someone@school.com", "role": "bursar"}
func (srv *Pool) ChangeAuthZHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.UserContextKey).(*auth.CustomClaims)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if claims.Role != auth.RoleAdmin {
		http.Error(w, "forbidden: insufficient privileges", http.StatusForbidden)
		return
	}
	ctx := r.Context()

	var req models.Role
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "400 invalid request body", http.StatusBadRequest)
		return
	}
	if !slices.Contains(knownRoles, req.Role) {
		http.Error(w, "400 role must be one of: admin, principal, teacher, bursar, parent", http.StatusBadRequest)
		return
	}
	if req.Email == claims.Email {
		// stops the last admin from demoting themselves and locking everyone out of this endpoint
		http.Error(w, "400 you can't change your own role", http.StatusBadRequest)
		return
	}

	updated, err := srv.Repo.UpdateRole(ctx, &req)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "404 user not found", http.StatusNotFound)
		return
	}
	if err != nil {
		slog.ErrorContext(ctx, "UpdateRole error", "error", err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, r, http.StatusOK, updated)
}
