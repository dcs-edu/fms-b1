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

	// INFO: FUCKING BOILERPLATE I DON'T EVEN UNDERSTAND PROPERLY
	srv := repository.Repository{DB : database}
	p := handlers.Pool{Repo: &srv}

	r := mux.NewRouter()
	r.Use(middleware.LoggingMiddleware)

	r.HandleFunc("/", handler).Methods("GET")
	r.HandleFunc("/auth/register", p.CreateUserHandler).Methods("POST")
	r.HandleFunc("/auth/login", p.LoginHandler).Methods("POST")

	subRouter := r.PathPrefix("/books").Subrouter()
	subRouter.Use(middleware.JwtMiddleware)

	subRouter.HandleFunc("/new", p.AddbookHandler).Methods("POST")

	// HACK: server port logging (remove if necessary)
	fmt.Printf("server running on port%v\n", port)
	http.ListenAndServe(port, r)

}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<h1>Welcome to my fuckass Go template backend code</h1><h3>Just modify a few things and Bob's your uncle.</h3>\nBTW,... You requested: %s\n This is your request struct btw:\n %v", r.URL.Path, r)
}
