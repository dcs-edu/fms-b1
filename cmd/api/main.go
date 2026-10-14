package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"

	"github.com/Jxt-Eli/template/internal/db"
	"github.com/Jxt-Eli/template/internal/handlers"
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
	ps := paystack.NewClient(secretKey)
	rpo := repository.NewRepository(database)
	p := handlers.NewPool(rpo, ps, jwtSecret)

	r := mux.NewRouter()
	r.Use(middleware.LoggingMiddleware)

	r.HandleFunc("/", handler).Methods("GET")
	r.HandleFunc("/auth/register", p.CreateUserHandler).Methods("POST")
	r.HandleFunc("/auth/login", p.LoginHandler).Methods("POST")
	// public on purpose: Paystack has no JWT, the signature header is what proves the call is real
	r.HandleFunc("/webhooks/paystack", p.PaystackWebhookHandler).Methods("POST")

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

	utilitiesRouter.HandleFunc("", p.CreateUtilityHandler).Methods("POST")
	utilitiesRouter.HandleFunc("/{util_id}/prices", p.CreateUtilityPriceHandler).Methods("POST")

	adminRouter := r.PathPrefix("/admin").Subrouter()
	adminRouter.Use(jwtCfg.JwtMiddleware)

	adminRouter.HandleFunc("/users/role", p.ChangeAuthZHandler).Methods("PATCH")

	parentsRouter := r.PathPrefix("/parents").Subrouter()
	parentsRouter.Use(jwtCfg.JwtMiddleware)

	parentsRouter.HandleFunc("/links", p.LinkParentHandler).Methods("POST")

	meRouter := r.PathPrefix("/me").Subrouter()
	meRouter.Use(jwtCfg.JwtMiddleware)

	meRouter.HandleFunc("/children", p.MyChildrenHandler).Methods("GET")

	// HACK: server port logging (remove if necessary)
	fmt.Printf("server running on port%v\n", port)
	// ListenAndServe only returns on failure (e.g. the port is already taken); without log.Fatal that error vanished
	log.Fatal(http.ListenAndServe(port, r))
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<h1>If you're seeing this, the backend works and the databse is running</h1><h3>How do I know? cause the backend won't start without a connection.</h3>")
}
