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
	"github.com/Jxt-Eli/template/internal/middleware"
	"github.com/Jxt-Eli/template/internal/handlers"
	"github.com/Jxt-Eli/template/internal/repository"
)

func main() {
	if err := godotenv.Load(); err != nil {
		slog.Error("Local env not found", "error", err)
	}
	dsn := os.Getenv("DB_URL")
	port := os.Getenv("PORT")
	database, err := db.Connect(dsn)
	if err != nil {
		log.Fatalf("Database connection failed: %v" ,err)
		return
	}
	defer database.Close()

	// boilerplate code that does the whole struct literal nesting drama nonsense
	// TODO: Consider replacing with constructor function call
	rpo := repository.Repository{DB : database}
	p := handlers.Pool{Repo: &rpo}

	r := mux.NewRouter()
	r.Use(middleware.LoggingMiddleware)

	r.HandleFunc("/", handler).Methods("GET")
	r.HandleFunc("/auth/register", p.CreateUserHandler).Methods("POST")
	r.HandleFunc("/auth/login", p.LoginHandler).Methods("POST")

	subRouter := r.PathPrefix("/new").Subrouter()
	subRouter.Use(middleware.JwtMiddleware)

	subRouter.HandleFunc("/book", p.AddbookHandler).Methods("POST")
	subRouter.HandleFunc("/student", p.StudentsHandler).Methods("POST")

	accountRouter := r.PathPrefix("/account").Subrouter()
	accountRouter.Use(middleware.JwtMiddleware)

	accountRouter.HandleFunc("/password", p.ChangePasswordHandler).Methods("PATCH")

	paymentsRouter := r.PathPrefix("/payments").Subrouter()
	paymentsRouter.Use(middleware.JwtMiddleware)

	paymentsRouter.HandleFunc("", p.PaymentHistoryHandler).Methods("GET")
	paymentsRouter.HandleFunc("", p.RecordPaymentHandler).Methods("POST")
	paymentsRouter.HandleFunc("/students/{student_id}", p.IndividualPaymentHistoryHandler).Methods("GET")

	adminRouter := r.PathPrefix("/admin").Subrouter()
	adminRouter.Use(middleware.JwtMiddleware)

	adminRouter.HandleFunc("/users/role", p.ChangeAuthZHandler).Methods("PATCH")

	parentsRouter := r.PathPrefix("/parents").Subrouter()
	parentsRouter.Use(middleware.JwtMiddleware)

	parentsRouter.HandleFunc("/links", p.LinkParentHandler).Methods("POST")

	meRouter := r.PathPrefix("/me").Subrouter()
	meRouter.Use(middleware.JwtMiddleware)

	meRouter.HandleFunc("/children", p.MyChildrenHandler).Methods("GET")

	// HACK: server port logging (remove if necessary)
	fmt.Printf("server running on port%v\n", port)
	http.ListenAndServe(port, r)

}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<h1>Welcome to my fuckass Go template backend code</h1><h3>Just modify a few things and Bob's your uncle.</h3>\nBTW,... You requested: %s\n This is your request struct btw:\n %v", r.URL.Path, r)
}
