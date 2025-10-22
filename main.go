package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"banking-backend/auth"

	_ "github.com/lib/pq"
)

func main() {
	// Get database connection details from environment variables
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		os.Getenv("DATABASE_HOST"), 5432, os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"), os.Getenv("POSTGRES_DB"))

	// Open a connection to the database
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Ping the database to verify the connection
	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Successfully connected to the database!")

	// Create the auth environment
	authEnv := &auth.Env{DB: db}

	// Create a new rate limiter
	rateLimiter := auth.NewRateLimiter()

	// Create a new ServeMux
	mux := http.NewServeMux()

	// Define handlers
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Welcome to the Banking System!")
	})

	// Auth routes
	mux.Handle("/signup", auth.ValidateSignupRequest(http.HandlerFunc(authEnv.SignupHandler)))
	mux.Handle("/login", rateLimiter.Middleware(http.HandlerFunc(authEnv.LoginHandler)))
	mux.Handle("/change-password", auth.AuthenticationMiddleware(http.HandlerFunc(authEnv.ChangePasswordHandler)))

	// Start the HTTP server
	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", auth.Logger(mux)); err != nil {
		log.Fatal(err)
	}
}
