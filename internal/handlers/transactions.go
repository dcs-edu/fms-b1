package handlers

import (
	"encoding/json"
	"database/sql"
	"net/http"
	"log/slog"
	"errors"
	"slices"
	"time"

	"github.com/gorilla/mux"

	"github.com/Jxt-Eli/template/internal/middleware"
	"github.com/Jxt-Eli/template/internal/repository"
	"github.com/Jxt-Eli/template/internal/models"
	"github.com/Jxt-Eli/template/internal/auth"
	"github.com/Jxt-Eli/template/pkg"
)

// paymentStaff are the roles that can see anyone's payments. Recording cash is narrower: see cashStaff.
var paymentStaff = []auth.Role{auth.RoleAdmin, auth.RolePrincipal, auth.RoleBursar}

// cashStaff are the roles that can record a payment no money went through Paystack for.
// The bursar is left out on purpose: they pay online like everyone else, so every payment they make has a Paystack trail.
var cashStaff = []auth.Role{auth.RoleAdmin, auth.RolePrincipal}

// GET /payments?range=this_month
func (srv *Pool) PaymentHistoryHandler(w http.ResponseWriter, r *http.Request) {
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

	// ResolveDateRange only fails on an unknown range key, so every error here is the client's fault
	start, end, err := pkg.ResolveDateRange(r.URL.Query().Get("range"), time.Now())
	if err != nil {
		http.Error(w, "400 range must be one of: today, this_week, this_month, three_months_to_date", http.StatusBadRequest)
		return
	}

	payments, err := srv.Repo.ListPaymentsInRange(ctx, start, end)
	if err != nil {
		slog.ErrorContext(ctx, "ListPaymentsInRange error", "error", err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, r, http.StatusOK, payments)
}

// GET /payments/students/{student_id}?range=this_month   e.g. /payments/students/30-0042?range=today
func (srv *Pool) IndividualPaymentHistoryHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.UserContextKey).(*auth.CustomClaims)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if claims.Role != auth.RoleParent && !slices.Contains(paymentStaff, claims.Role) {
		http.Error(w, "forbidden: insufficient privileges", http.StatusForbidden)
		return
	}
	ctx := r.Context()

	gradYear, seqNum, err := pkg.ParseStudentID(mux.Vars(r)["student_id"])
	if err != nil {
		http.Error(w, "400 "+err.Error(), http.StatusBadRequest)
		return
	}

	// ResolveDateRange only fails on an unknown range key, so every error here is the client's fault
	start, end, err := pkg.ResolveDateRange(r.URL.Query().Get("range"), time.Now())
	if err != nil {
		http.Error(w, "400 range must be one of: today, this_week, this_month, three_months_to_date", http.StatusBadRequest)
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

	// A parent only sees their own children. An unlinked child gets the same 404 as a missing one,
	// so a parent can't probe which student IDs exist.
	if claims.Role == auth.RoleParent {
		linked, err := srv.Repo.IsParentOf(ctx, claims.ID, std.AdmissionNo)
		if err != nil {
			slog.ErrorContext(ctx, "IsParentOf error", "error", err)
			http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
			return
		}
		if !linked {
			http.Error(w, "404 student not found", http.StatusNotFound)
			return
		}
	}

	payments, err := srv.Repo.GetIndividualPaymentsInRange(ctx, start, end, std.AdmissionNo)
	if err != nil {
		slog.ErrorContext(ctx, "GetIndividualPaymentsInRange error", "error", err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}
	// which user started each payment is internal (it can be a staff member); parents don't see it
	if claims.Role == auth.RoleParent {
		for i := range payments {
			payments[i].InitiatedBy = nil
		}
	}
	writeJSON(w, r, http.StatusOK, models.StudentPayments{Student: *std, Payments: payments})
}

// POST /payments   {"student_id": "30-0042", "bill_id": 3, "amount": "150.00"}
// An admin or principal recording a cash payment. Who recorded it comes from their token, not the body.
func (srv *Pool) RecordPaymentHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.UserContextKey).(*auth.CustomClaims)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if !slices.Contains(cashStaff, claims.Role) {
		http.Error(w, "forbidden: insufficient privileges", http.StatusForbidden)
		return
	}
	ctx := r.Context()

	var req models.NewPayment
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "400 invalid request body", http.StatusBadRequest)
		return
	}

	gradYear, seqNum, err := pkg.ParseStudentID(req.StudentID)
	if err != nil {
		http.Error(w, "400 "+err.Error(), http.StatusBadRequest)
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

	payment, err := srv.Repo.InsertPayment(ctx, models.Payment{
		BillID:      req.BillID,
		AdmissionNo: std.AdmissionNo,
		Amount:      req.Amount,
		InitiatedBy: &claims.ID,
	})
	switch {
	case errors.Is(err, repository.ErrNonPositiveAmount), errors.Is(err, repository.ErrOverpayment):
		http.Error(w, "400 "+err.Error(), http.StatusBadRequest)
		return
	case errors.Is(err, repository.ErrBillNotFound), errors.Is(err, repository.ErrStudentNotFound):
		http.Error(w, "404 "+err.Error(), http.StatusNotFound)
		return
	case err != nil:
		slog.ErrorContext(ctx, "InsertPayment error", "error", err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, r, http.StatusCreated, payment)
}

// writeJSON sets headers before WriteHeader on purpose: once the status is sent, header changes are ignored.
func writeJSON(w http.ResponseWriter, r *http.Request, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.ErrorContext(r.Context(), "json encode error", "error", err)
	}
}
