package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"

	"github.com/Jxt-Eli/template/internal/db"
	"github.com/Jxt-Eli/template/internal/handlers"
	"github.com/Jxt-Eli/template/internal/mailer"
	"github.com/Jxt-Eli/template/internal/middleware"
	"github.com/Jxt-Eli/template/internal/paystack"
	"github.com/Jxt-Eli/template/internal/repository"
)

func main() {
	if err := godotenv.Load(); err != nil {
		slog.Error("Local env not found", "error", err)
	}
	dsn := os.Getenv("DB_URL")
	port := os.Getenv("PORT")
	secretKey := os.Getenv("PAYSTACK_SECRET_KEY")
	jwtSecret := []byte(os.Getenv("JWTSECRET"))
	// optional: the public address of GET /payments/callback. Empty means Paystack uses its dashboard setting.
	callbackURL := os.Getenv("PAYSTACK_CALLBACK_URL")
	// optional: with SMTP_HOST empty, emails (reset codes) are printed to the log instead of sent. Dev only.
	ml := mailer.New(os.Getenv("SMTP_HOST"), os.Getenv("SMTP_PORT"), os.Getenv("SMTP_USER"),
		os.Getenv("SMTP_PASS"), os.Getenv("SMTP_FROM"))

	// an empty JWTSECRET would sign every token with an empty key, which anyone can forge
	if dsn == "" || port == "" || secretKey == "" || len(jwtSecret) == 0 {
		log.Fatal("error: missing port, dsn, paystack secret, or jwt secret")
	}
	jwtCfg := middleware.Config{JWTSecret: jwtSecret}

	database, err := db.Connect(dsn)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
		return
	}
	defer database.Close()

	// constructor function ceremony
	ps := paystack.NewClient(secretKey, callbackURL)
	rpo := repository.NewRepository(database)
	p := handlers.NewPool(rpo, ps, ml, jwtSecret)

	r := mux.NewRouter()
	r.Use(middleware.LoggingMiddleware)
	r.Use(middleware.MaxBodyMiddleware)

	r.HandleFunc("/", handler).Methods("GET")
	r.HandleFunc("/healthz", p.HealthHandler).Methods("GET")
	r.HandleFunc("/auth/register", p.CreateUserHandler).Methods("POST")
	r.HandleFunc("/auth/login", p.LoginHandler).Methods("POST")
	// public: someone who forgot their password has no token. The emailed code is what proves who they are.
	r.HandleFunc("/auth/forgot-password", p.ForgotPasswordHandler).Methods("POST")
	r.HandleFunc("/auth/reset-password", p.ResetPasswordHandler).Methods("POST")
	// public on purpose: Paystack has no JWT, the signature header is what proves the call is real
	r.HandleFunc("/webhooks/paystack", p.PaystackWebhookHandler).Methods("POST")
	// public too: it's the payer's browser landing here after checkout, and a browser has no JWT
	r.HandleFunc("/payments/callback", p.PaystackCallbackHandler).Methods("GET")

	subRouter := r.PathPrefix("/new").Subrouter()
	subRouter.Use(jwtCfg.JwtMiddleware)

	subRouter.HandleFunc("/book", p.AddbookHandler).Methods("POST")
	subRouter.HandleFunc("/student", p.StudentsHandler).Methods("POST")

	accountRouter := r.PathPrefix("/account").Subrouter()
	accountRouter.Use(jwtCfg.JwtMiddleware)

	accountRouter.HandleFunc("/password", p.ChangePasswordHandler).Methods("PATCH")

	paymentsRouter := r.PathPrefix("/payments").Subrouter()
	paymentsRouter.Use(jwtCfg.JwtMiddleware)

	paymentsRouter.HandleFunc("", p.PaymentHistoryHandler).Methods("GET")
	paymentsRouter.HandleFunc("", p.RecordPaymentHandler).Methods("POST")
	paymentsRouter.HandleFunc("/students/{student_id}", p.IndividualPaymentHistoryHandler).Methods("GET")
	paymentsRouter.HandleFunc("/online", p.OnlinePaymentHandler).Methods("POST")
	paymentsRouter.HandleFunc("/online/{reference}", p.VerifyOnlinePaymentHandler).Methods("GET")

	utilitiesRouter := r.PathPrefix("/utilities").Subrouter()
	utilitiesRouter.Use(jwtCfg.JwtMiddleware)

	// same path, different methods: GET reads the list, POST adds to it
	utilitiesRouter.HandleFunc("/create", p.CreateUtilityHandler).Methods("POST")
	utilitiesRouter.HandleFunc("/list", p.ListUtilityHandler).Methods("GET")
	utilitiesRouter.HandleFunc("/{util_id}/prices", p.CreateUtilityPriceHandler).Methods("POST")
	utilitiesRouter.HandleFunc("/remove/{id}", p.RemoveUtilityHandler).Methods("PATCH")

	studentsRouter := r.PathPrefix("/students").Subrouter()
	studentsRouter.Use(jwtCfg.JwtMiddleware)

	studentsRouter.HandleFunc("", p.ListStudentsHandler).Methods("GET")
	studentsRouter.HandleFunc("/profile/{student_id}", p.FullProfileHandler).Methods("GET")
	studentsRouter.HandleFunc("/{student_id}/bills", p.StudentBillsHandler).Methods("GET")
	studentsRouter.HandleFunc("/owing", p.StudentDebtHandler).Methods("GET")

	adminRouter := r.PathPrefix("/admin").Subrouter()
	adminRouter.Use(jwtCfg.JwtMiddleware)

	adminRouter.HandleFunc("/users", p.ShowUserRolesHandler).Methods("GET")
	adminRouter.HandleFunc("/users/role", p.ChangeAuthZHandler).Methods("PATCH")

	parentsRouter := r.PathPrefix("/parents").Subrouter()
	parentsRouter.Use(jwtCfg.JwtMiddleware)

	parentsRouter.HandleFunc("/links", p.LinkParentHandler).Methods("POST")

	meRouter := r.PathPrefix("/me").Subrouter()
	meRouter.Use(jwtCfg.JwtMiddleware)

	meRouter.HandleFunc("", p.ProfileHandler).Methods("GET")
	meRouter.HandleFunc("/children", p.MyChildrenHandler).Methods("GET")

	server := &http.Server{
		Addr:              port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		// longer than the Paystack client's 10s timeout: OnlinePaymentHandler can wait that long on Paystack
		// and still needs time to answer. A WriteTimeout shorter than that cuts the reply off mid-payment.
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown. `docker stop` and Ctrl+C send a signal; without this the process died on the spot,
	// possibly halfway through saving a payment. NotifyContext gives a ctx that is cancelled when SIGINT
	// (Ctrl+C) or SIGTERM (docker stop) arrives.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// ListenAndServe blocks until the server stops, so it runs in its own goroutine and main waits for the signal below.
	go func() {
		// HACK: server port logging (remove if necessary)
		fmt.Printf("server running on port%v\n", port)
		// ErrServerClosed is what ListenAndServe returns after Shutdown: expected, not a failure
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	slog.Info("shutdown signal received, finishing in-flight requests")

	// Shutdown stops taking new connections and waits for running requests to finish. Docker waits 10s
	// before sending SIGKILL, so give up a little before that.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
	}
	// main returns here, so the deferred database.Close() finally runs (log.Fatal skipped it before)
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<h1>If you're seeing this, the backend works and the database is running</h1><h3>How do I know? cause the backend won't start without a connection.</h3>")
}
