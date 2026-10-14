package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/shopspring/decimal"

	"github.com/Jxt-Eli/template/internal/auth"
	"github.com/Jxt-Eli/template/internal/middleware"
	"github.com/Jxt-Eli/template/internal/models"
	"github.com/Jxt-Eli/template/internal/repository"
)

// POST /utilities   {"util_name": "Tuition"}
func (srv *Pool) CreateUtilityHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.UserContextKey).(*auth.CustomClaims)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if !slices.Contains(paymentStaff, claims.Role) {
		http.Error(w, "forbidden: insufficient privileges", http.StatusForbidden)
		return
	}
	ctx := r.Context()

	var req struct {
		UtilName string `json:"util_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "400 invalid request body", http.StatusBadRequest)
		return
	}
	req.UtilName = strings.TrimSpace(req.UtilName)
	if req.UtilName == "" {
		http.Error(w, "400 util_name is required", http.StatusBadRequest)
		return
	}

	// A new utility always starts active. Taking is_active from the body would turn a missing
	// field into Go's zero value, false, the same trap as the zero admission_no in InsertStudent.
	utility, err := srv.Repo.InsertUtility(ctx, models.Utility{UtilName: req.UtilName, IsActive: true})
	if err != nil {
		slog.ErrorContext(ctx, "InsertUtility error", "error", err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, r, http.StatusCreated, utility)
}

// POST /utilities/{util_id}/prices   {"sem": "2026-T1", "amount": "500.00"}
// The new price is a bill: every student owes it for that semester.
func (srv *Pool) CreateUtilityPriceHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.UserContextKey).(*auth.CustomClaims)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if !slices.Contains(paymentStaff, claims.Role) {
		http.Error(w, "forbidden: insufficient privileges", http.StatusForbidden)
		return
	}
	ctx := r.Context()

	utilID, err := strconv.ParseInt(mux.Vars(r)["util_id"], 10, 64)
	if err != nil {
		http.Error(w, "400 util_id must be a number", http.StatusBadRequest)
		return
	}

	var req struct {
		Sem    string          `json:"sem"`
		Amount decimal.Decimal `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "400 invalid request body", http.StatusBadRequest)
		return
	}
	req.Sem = strings.TrimSpace(req.Sem)
	if req.Sem == "" {
		http.Error(w, "400 sem is required", http.StatusBadRequest)
		return
	}

	price, err := srv.Repo.InsertUtilityPrice(ctx, models.UtilityPrice{UtilID: utilID, Sem: req.Sem, Amount: req.Amount})
	switch {
	case errors.Is(err, repository.ErrNonPositiveAmount):
		http.Error(w, "400 "+err.Error(), http.StatusBadRequest)
		return
	case errors.Is(err, repository.ErrInvalidReference):
		http.Error(w, "404 utility not found", http.StatusNotFound)
		return
	case errors.Is(err, repository.ErrDuplicate):
		http.Error(w, "409 this utility already has a price for that semester", http.StatusConflict)
		return
	case err != nil:
		slog.ErrorContext(ctx, "InsertUtilityPrice error", "error", err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, r, http.StatusCreated, price)
}
