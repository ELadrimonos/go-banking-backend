package main

import (
	"banking-backend/account"
	"banking-backend/auth"
	"banking-backend/currency"
	"banking-backend/transactions"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

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
	accountEnv := &account.Env{DB: db}
	transactionsEnv := &transactions.Env{DB: db}

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
	mux.Handle("/refresh", http.HandlerFunc(authEnv.RefreshHandler))
	mux.Handle("/status", auth.AuthenticationMiddleware(http.HandlerFunc(authEnv.StatusHandler)))

	// Account routes
	mux.Handle("/accounts", auth.AuthenticationMiddleware(http.HandlerFunc(accountEnv.GetAccountsHandler)))
	mux.Handle("/create-account", auth.AuthenticationMiddleware(http.HandlerFunc(accountEnv.CreateAccountHandler)))

	// Transactions routes
	mux.Handle("/deposit", auth.AuthenticationMiddleware(http.HandlerFunc(transactionsEnv.DepositHandler)))

	// Currency conversion route
	mux.HandleFunc("/convert", func(w http.ResponseWriter, r *http.Request) {
		from := r.URL.Query().Get("from")
		to := r.URL.Query().Get("to")
		amountStr := r.URL.Query().Get("amount")

		if from == "" || to == "" || amountStr == "" {
			http.Error(w, "Missing required query parameters: from, to, amount", http.StatusBadRequest)
			return
		}

		amount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil {
			http.Error(w, "Invalid amount", http.StatusBadRequest)
			return
		}

		rate, err := currency.GetRate(from, to)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get exchange rate: %v", err), http.StatusInternalServerError)
			return
		}

		convertedAmount := amount * rate
		fmt.Fprintf(w, "%.2f %s is %.2f %s", amount, from, convertedAmount, to)
	})

	// Start the HTTP server
	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", auth.Logger(mux)); err != nil {
		log.Fatal(err)
	}
}
