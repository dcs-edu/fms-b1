package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"slices"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"github.com/Jxt-Eli/template/internal/auth"
	"github.com/Jxt-Eli/template/internal/middleware"
	"github.com/Jxt-Eli/template/internal/models"
	"github.com/Jxt-Eli/template/internal/paystack"
	"github.com/Jxt-Eli/template/internal/repository"
	"github.com/Jxt-Eli/template/pkg"
)

// POST /payments/online   {"student_id": "30-0042", "bill_id": 3, "amount": "150.00"}
// A parent (for their own child) or the bursar (for any student) starts paying a bill.
// The reply has the Paystack page to send them to.
func (srv *Pool) OnlinePaymentHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.UserContextKey).(*auth.CustomClaims)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if claims.Role != auth.RoleParent && claims.Role != auth.RoleBursar {
		http.Error(w, "forbidden: only parents and the bursar pay online", http.StatusForbidden)
		return
	}
	ctx := r.Context()

	var req models.NewPayment
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "400 invalid request body", http.StatusBadRequest)
		return
	}
	if !req.Amount.IsPositive() {
		http.Error(w, "400 "+repository.ErrNonPositiveAmount.Error(), http.StatusBadRequest)
		return
	}
	// Paystack takes whole pesewas. ToCoin would quietly chop "150.005" down to 15000,
	// so the parent would be charged less than the row says. Refuse it instead.
	if !req.Amount.Equal(req.Amount.Round(2)) {
		http.Error(w, "400 amount can have at most 2 decimal places", http.StatusBadRequest)
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

	// A parent only pays for their own children; the bursar can pay for anyone.
	// Same 404 as a missing student, so a parent can't probe which student IDs exist.
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

	remaining, err := srv.Repo.RemainingOnBill(ctx, std.AdmissionNo, req.BillID)
	if errors.Is(err, repository.ErrBillNotFound) {
		http.Error(w, "404 "+err.Error(), http.StatusNotFound)
		return
	}
	if err != nil {
		slog.ErrorContext(ctx, "RemainingOnBill error", "error", err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}
	if req.Amount.GreaterThan(remaining) {
		http.Error(w, "400 "+repository.ErrOverpayment.Error()+": "+remaining.StringFixed(2)+" left to pay", http.StatusBadRequest)
		return
	}

	// The pending row is saved BEFORE talking to Paystack, so when the webhook arrives
	// there is always a row with this reference waiting for it.
	reference := uuid.NewString()
	payment, err := srv.Repo.InsertPendingPayment(ctx, models.Payment{
		BillID:      req.BillID,
		AdmissionNo: std.AdmissionNo,
		Amount:      req.Amount,
		Reference:   &reference,
		InitiatedBy: &claims.ID,
	})
	if err != nil {
		slog.ErrorContext(ctx, "InsertPendingPayment error", "error", err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}

	authURL, err := srv.Paystack.InitializeTransaction(ctx, paystack.ToCoin(req.Amount), claims.Email, reference)
	if err != nil {
		slog.ErrorContext(ctx, "InitializeTransaction error", "error", err, "reference", reference)
		// The parent never got a payment page, so this attempt can't be paid. Mark it failed so it
		// doesn't sit in "pending" forever. If Paystack somehow took money anyway, CompletePayment still wins.
		if _, err := srv.Repo.FailPayment(ctx, reference); err != nil {
			slog.ErrorContext(ctx, "FailPayment error", "error", err, "reference", reference)
		}
		// 502 Bad Gateway: our server is fine, the service we depend on failed
		http.Error(w, "502 could not reach the payment provider, try again", http.StatusBadGateway)
		return
	}

	writeJSON(w, r, http.StatusCreated, struct {
		Payment          *models.Payment `json:"payment"`
		AuthorizationURL string          `json:"authorization_url"`
	}{payment, authURL})
}

// GET /payments/online/{reference}
// Asks Paystack how a payment went. The parent's page calls this after Paystack sends them back.
// It is also the only way a payment becomes "failed": Paystack sends no webhook for failed charges.
func (srv *Pool) VerifyOnlinePaymentHandler(w http.ResponseWriter, r *http.Request) {
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
	reference := mux.Vars(r)["reference"]

	payment, err := srv.Repo.GetPaymentByReference(ctx, reference)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "404 payment not found", http.StatusNotFound)
		return
	}
	if err != nil {
		slog.ErrorContext(ctx, "GetPaymentByReference error", "error", err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}

	if claims.Role == auth.RoleParent {
		linked, err := srv.Repo.IsParentOf(ctx, claims.ID, payment.AdmissionNo)
		if err != nil {
			slog.ErrorContext(ctx, "IsParentOf error", "error", err)
			http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
			return
		}
		if !linked {
			http.Error(w, "404 payment not found", http.StatusNotFound)
			return
		}
	}

	// Completed is final, so there is nothing to ask Paystack. Failed is asked again,
	// because a declined card can be retried on the same Paystack page.
	if payment.Status != models.PaymentCompleted {
		res, err := srv.Paystack.VerifyTransaction(ctx, reference)
		if err != nil {
			slog.ErrorContext(ctx, "VerifyTransaction error", "error", err, "reference", reference)
			http.Error(w, "502 could not reach the payment provider, try again", http.StatusBadGateway)
			return
		}

		// Paystack's statuses: success, failed, abandoned (page opened, never paid), and in-between
		// ones like ongoing or pending. "abandoned" is left as pending: the parent can still go back
		// and pay, and calling it failed would just be wrong for a while.
		switch res.Data.Status {
		case "success":
			// Never trust "success" alone: check Paystack took the amount this row is for.
			if res.Data.Amount != paystack.ToCoin(payment.Amount) {
				slog.ErrorContext(ctx, "paystack amount does not match payment", "reference", reference,
					"expected", paystack.ToCoin(payment.Amount), "got", res.Data.Amount)
				http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
				return
			}
			if _, err := srv.Repo.CompletePayment(ctx, reference); err != nil {
				slog.ErrorContext(ctx, "CompletePayment error", "error", err)
				http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
				return
			}
		case "failed":
			if _, err := srv.Repo.FailPayment(ctx, reference); err != nil {
				slog.ErrorContext(ctx, "FailPayment error", "error", err)
				http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
				return
			}
		}

		// read it back so the reply shows the new status and paid_at
		payment, err = srv.Repo.GetPaymentByReference(ctx, reference)
		if err != nil {
			slog.ErrorContext(ctx, "GetPaymentByReference error", "error", err)
			http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
			return
		}
	}
	// same rule as the payment history: parents don't see which user started a payment
	if claims.Role == auth.RoleParent {
		payment.InitiatedBy = nil
	}
	writeJSON(w, r, http.StatusOK, payment)
}

// POST /webhooks/paystack
// Paystack calls this when a charge succeeds. Paystack retries until it gets a 2xx back.
func (srv *Pool) PaystackWebhookHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Keep the raw bytes: the signature was computed over exactly these bytes, so they must be
	// checked before decoding. MaxBytesReader stops a stranger from sending a 5GB body.
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		http.Error(w, "400 could not read body", http.StatusBadRequest)
		return
	}
	if !srv.Paystack.VerifySignature(body, r.Header.Get("x-paystack-signature")) {
		slog.WarnContext(ctx, "paystack webhook with bad signature", "remote", r.RemoteAddr)
		http.Error(w, "401 invalid signature", http.StatusUnauthorized)
		return
	}

	var event struct {
		Event string `json:"event"` // e.g. "charge.success"
		Data  struct {
			Reference string `json:"reference"`
			Amount    int64  `json:"amount"`   // pesewas
			Currency  string `json:"currency"` // "GHS"
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &event); err != nil {
		http.Error(w, "400 invalid body", http.StatusBadRequest)
		return
	}

	// One question decides every status code below: "would Paystack sending this again help?"
	// Yes (our DB had a hiccup) -> 500, Paystack retries later. No -> 200, so it stops.
	// A 200 here means "message received", not "payment completed".
	switch event.Event {
	case "charge.success":
		payment, err := srv.Repo.GetPaymentByReference(ctx, event.Data.Reference)
		if errors.Is(err, sql.ErrNoRows) {
			// A reference we never created, e.g. a payment made from the Paystack dashboard.
			// Sending it again won't make the row appear.
			slog.WarnContext(ctx, "paystack webhook for unknown reference", "reference", event.Data.Reference)
			w.WriteHeader(http.StatusOK)
			return
		}
		if err != nil {
			slog.ErrorContext(ctx, "GetPaymentByReference error", "error", err)
			http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
			return
		}

		// The signature proves Paystack sent this, not that the parent paid the right amount.
		// A mismatch stays pending for a person to look at: a retry would carry the same numbers.
		if event.Data.Amount != paystack.ToCoin(payment.Amount) || event.Data.Currency != "GHS" {
			slog.ErrorContext(ctx, "paystack amount does not match payment", "reference", event.Data.Reference,
				"expected", paystack.ToCoin(payment.Amount), "got", event.Data.Amount, "currency", event.Data.Currency)
			w.WriteHeader(http.StatusOK)
			return
		}

		changed, err := srv.Repo.CompletePayment(ctx, event.Data.Reference)
		if err != nil {
			slog.ErrorContext(ctx, "CompletePayment error", "error", err)
			http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
			return
		}
		if !changed {
			// already completed: a repeat delivery, or the verify endpoint got there first
			slog.InfoContext(ctx, "paystack webhook repeated", "reference", event.Data.Reference)
		}
		w.WriteHeader(http.StatusOK)

	default:
		// Events this app doesn't use (transfers, subscriptions, ...). Acknowledge them so Paystack
		// doesn't keep retrying something we will never act on.
		w.WriteHeader(http.StatusOK)
	}
}
