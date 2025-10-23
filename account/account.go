package account

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"banking-backend/auth"
)

// --- Models ---

type Account struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	AccountNumber string    `json:"account_number"`
	Balance       float64   `json:"balance"`
	Currency      string    `json:"currency"`
	AccountType   string    `json:"account_type"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type CreateAccountRequest struct {
	AccountType string `json:"account_type"`
	Currency    string `json:"currency"`
}

// --- Database ---

type DB struct {
	*sql.DB
}

func (db *DB) CreateAccount(ctx context.Context, account *Account) (string, error) {
	var id string
	query := `INSERT INTO accounts (user_id, account_number, balance, currency, account_type)
			  VALUES ($1, $2, $3, $4, $5) RETURNING id`
	err := db.QueryRowContext(ctx, query, account.UserID, account.AccountNumber, account.Balance, account.Currency, account.AccountType).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("could not create account: %w", err)
	}
	return id, nil
}

func (db *DB) GetAccountsByUserID(userID string) ([]*Account, error) {
	rows, err := db.Query(`SELECT id, user_id, account_number, balance, currency, account_type, created_at, updated_at
					   FROM accounts WHERE user_id = $1`, userID)
	if err != nil {
		return nil, fmt.Errorf("could not get accounts by user id: %w", err)
	}
	defer rows.Close()

	var accounts []*Account
	for rows.Next() {
		account := &Account{}
		err := rows.Scan(&account.ID, &account.UserID, &account.AccountNumber, &account.Balance, &account.Currency, &account.AccountType, &account.CreatedAt, &account.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("could not scan account: %w", err)
		}
		accounts = append(accounts, account)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating accounts: %w", err)
	}

	return accounts, nil
}

// --- Handlers ---

type Env struct {
	DB *sql.DB
}

func (env *Env) CreateAccountHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r)
	if err != nil {
		auth.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req CreateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		auth.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.AccountType == "" {
		req.AccountType = "checking"
	}
	if req.Currency == "" {
		req.Currency = "USD"
	}

	accountNumber, err := generateAccountNumber()
	if err != nil {
		auth.RespondWithError(w, http.StatusInternalServerError, "Failed to generate account number")
		return
	}

	db := &DB{env.DB}
	account := &Account{
		UserID:        userID,
		AccountNumber: accountNumber,
		Balance:       0,
		Currency:      req.Currency,
		AccountType:   req.AccountType,
	}

	accountID, err := db.CreateAccount(r.Context(), account)
	if err != nil {
		auth.RespondWithError(w, http.StatusInternalServerError, "Failed to create account")
		return
	}

	auth.JSON(w, http.StatusCreated, map[string]string{"account_id": accountID, "account_number": accountNumber})
}

func (env *Env) GetAccountsHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r)
	if err != nil {
		auth.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	db := &DB{env.DB}
	accounts, err := db.GetAccountsByUserID(userID)
	if err != nil {
		auth.RespondWithError(w, http.StatusInternalServerError, "Failed to get accounts")
		return
	}

	if len(accounts) == 0 {
		auth.JSON(w, http.StatusOK, []*Account{})
		return
	}

	auth.JSON(w, http.StatusOK, accounts)
}

func generateAccountNumber() (string, error) {
	// Generate a random 10-digit account number
	var b [10]byte
	_, err := rand.Read(b[:])
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d", b),
		nil
}
