package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"log/slog"
	"slices"

	"github.com/gorilla/mux"

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
		StudentID: pkg.FormatStudentID(newStd.GradYear, newStd.SeqNum),
	})
}

func (srv *Pool) FullProfileHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	claims, ok := r.Context().Value(middleware.UserContextKey).(*auth.CustomClaims)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if !slices.Contains(paymentStaff, claims.Role) {
		http.Error(w, "forbidden: insufficient privileges", http.StatusForbidden)
		return
	}
	var studentProfile models.Student
	studentProfile.SeqNum, studentProfile.GradYear, _ = pkg.ParseStudentID(studentProfile.StudentID)
	profile, err := srv.Repo.FullStudentProfile(ctx, &studentProfile)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w,"student profile not found", http.StatusNotFound)
		return
	}
	if err != nil {
		slog.Error("FullProfileHandler error:", "error", err)
		http.Error(w, "500 internal server error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, r, http.StatusOK, profile)
}
// GET /students
// Every student, for staff. Parents use GET /me/children instead, which only lists their own.
func (srv *Pool) ListStudentsHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.UserContextKey).(*auth.CustomClaims)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if claims.Role == auth.RoleParent {
		http.Error(w, "forbidden: insufficient privileges", http.StatusForbidden)
		return
	}
	ctx := r.Context()

	students, err := srv.Repo.GetAllStudents(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "GetAllStudents error", "error", err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, r, http.StatusOK, students)
}

// GET /students/{student_id}/bills   e.g. /students/30-0042/bills
// What a student owes: every bill, what has been paid on it, and what is left. This is the parent's
// main screen, and where the app gets the bill_id to send to POST /payments/online.
func (srv *Pool) StudentBillsHandler(w http.ResponseWriter, r *http.Request) {
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

	// same rule as the payment history: a parent only sees their own children,
	// and an unlinked child gets the same 404 as a missing one
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

	bills, err := srv.Repo.StudentBills(ctx, std.AdmissionNo)
	if err != nil {
		slog.ErrorContext(ctx, "StudentBills error", "error", err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, r, http.StatusOK, struct {
		Student models.Std           `json:"student"`
		Bills   []models.StudentBill `json:"bills"`
	}{*std, bills})
}

// GET /students/owing
// Every student with money left on a bill, one row per unfinished bill.
func (srv *Pool) StudentDebtHandler(w http.ResponseWriter, r *http.Request) {
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

	debts, err := srv.Repo.ShowStudentDebts(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "ShowStudentDebts error", "error", err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, r, http.StatusOK, debts)
}
