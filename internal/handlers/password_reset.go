package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/Jxt-Eli/template/internal/models"
)

// The same reply whether the email has an account or not, so this endpoint can't be used to find out who
// has one. (POST /auth/register still tells, with its 409. That one is hard to avoid on a sign-up form.)
const forgotPasswordReply = "if an account with that email exists, a reset code has been sent to it"

// POST /auth/forgot-password   {"email": "mum@example.com"}
// Emails the user an 8-digit code. The app then shows a "code + new password" screen,
// which calls POST /auth/reset-password. A code, not a link: a parent types it into the app, so the
// app needs no deep-link setup.
func (srv *Pool) ForgotPasswordHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req models.ForgotPassword
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "400 invalid request body", http.StatusBadRequest)
		return
	}

	user, err := srv.Repo.GetByEmail(ctx, strings.TrimSpace(req.Email))
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, r, http.StatusAccepted, map[string]string{"message": forgotPasswordReply})
		return
	}
	if err != nil {
		slog.ErrorContext(ctx, "GetByEmail error", "error", err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}

	// crypto/rand, never math/rand: math/rand's numbers can be predicted, and a predictable code is a stolen account.
	// 8 digits = 100 million codes. With 5 guesses per code and a new code at most once a minute,
	// guessing one would take a lifetime.
	n, err := rand.Int(rand.Reader, big.NewInt(100_000_000))
	if err != nil {
		slog.ErrorContext(ctx, "reset code error", "error", err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}
	code := fmt.Sprintf("%08d", n.Int64()) // %08d keeps leading zeros: 42 becomes "00000042"

	saved, err := srv.Repo.SavePasswordReset(ctx, user.ID, hashResetCode(code))
	if err != nil {
		slog.ErrorContext(ctx, "SavePasswordReset error", "error", err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}
	if !saved {
		// asked again within a minute: the code already sent still works, so send nothing new
		writeJSON(w, r, http.StatusAccepted, map[string]string{"message": forgotPasswordReply})
		return
	}

	body := "Hello " + user.Name + ",\n\n" +
		"Your password reset code is: " + code + "\n\n" +
		"It expires in 15 minutes. If you didn't ask for this, ignore this email; your password hasn't changed.\n"
	if err := srv.Mailer.Send(user.Email, "Your password reset code", body); err != nil {
		slog.ErrorContext(ctx, "send reset email error", "error", err)
		// 502: our server is fine, the mail server isn't. Same idea as Paystack failing.
		http.Error(w, "502 could not send the email, try again in a minute", http.StatusBadGateway)
		return
	}
	writeJSON(w, r, http.StatusAccepted, map[string]string{"message": forgotPasswordReply})
}

// POST /auth/reset-password   {"email", "code", "new_password"}
// Every way the code can be wrong (no code, expired, used up, mistyped) gets the same answer,
// so a guesser learns nothing from the replies.
func (srv *Pool) ResetPasswordHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req models.ResetPassword
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "400 invalid request body", http.StatusBadRequest)
		return
	}
	// checked before the code, so a typo in the new password doesn't burn one of the 5 guesses
	if len(req.NewPassword) < minPasswordLen {
		http.Error(w, "400 new password must be at least 8 characters", http.StatusBadRequest)
		return
	}
	if len(req.NewPassword) > 72 {
		http.Error(w, "400 new password must be at most 72 bytes", http.StatusBadRequest)
		return
	}

	user, err := srv.Repo.GetByEmail(ctx, strings.TrimSpace(req.Email))
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "400 invalid or expired code", http.StatusBadRequest)
		return
	}
	if err != nil {
		slog.ErrorContext(ctx, "GetByEmail error", "error", err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}

	storedHash, err := srv.Repo.UseResetAttempt(ctx, user.ID)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "400 invalid or expired code", http.StatusBadRequest)
		return
	}
	if err != nil {
		slog.ErrorContext(ctx, "UseResetAttempt error", "error", err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}

	// ConstantTimeCompare takes as long for a near miss as for a total miss. A plain == stops at the
	// first differing byte, and that tiny timing difference can leak the answer one byte at a time.
	codeHash := hashResetCode(strings.TrimSpace(req.Code))
	if subtle.ConstantTimeCompare([]byte(codeHash), []byte(storedHash)) != 1 {
		http.Error(w, "400 invalid or expired code", http.StatusBadRequest)
		return
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		slog.ErrorContext(ctx, "password hash error", "error", err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}

	err = srv.Repo.ResetPassword(ctx, user.ID, codeHash, string(newHash))
	if errors.Is(err, sql.ErrNoRows) {
		// another request used this code a moment ago
		http.Error(w, "400 invalid or expired code", http.StatusBadRequest)
		return
	}
	if err != nil {
		slog.ErrorContext(ctx, "ResetPassword error", "error", err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// hashResetCode is what gets stored instead of the code, so someone reading the database (or a backup)
// can't use a live code. sha256 rather than bcrypt: the code is random and dies after 15 minutes and
// 5 guesses, so it doesn't need bcrypt's deliberate slowness the way a password chosen by a person does.
func hashResetCode(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}
