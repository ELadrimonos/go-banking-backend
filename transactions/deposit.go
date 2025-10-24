package transactions

import (
	"banking-backend/account"
	"banking-backend/auth"
	"banking-backend/currency"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
)

// --- Models ---

type DepositRequest struct {
	AccountNumber string  `json:"account_number"`
	Amount        float64 `json:"amount"`
	Currency      string  `json:"currency"`
}

// --- Handlers ---

type Env struct {
	DB *sql.DB
}

func (env *Env) DepositHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r)
	if err != nil {
		auth.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req DepositRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		auth.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Amount <= 0 {
		auth.RespondWithError(w, http.StatusBadRequest, "Deposit amount must be positive")
		return
	}

	// Get the account from the database
	db := &account.DB{env.DB}
	acc, err := db.GetAccountByAccountNumber(req.AccountNumber)
	if err != nil || acc == nil {
		auth.RespondWithError(w, http.StatusNotFound, "Account not found")
		return
	}

	// Check if the account belongs to the user
	if acc.UserID != userID {
		auth.RespondWithError(w, http.StatusUnauthorized, "Account does not belong to the user")
		return
	}

	depositedAmount := req.Amount
	// Convert currency if necessary
	if req.Currency != acc.Currency {
		rate, err := currency.GetRate(req.Currency, acc.Currency)
		if err != nil {
			auth.RespondWithError(w, http.StatusInternalServerError, "Failed to get exchange rate")
			return
		}
		depositedAmount = req.Amount * rate
	}

	// Update the account balance
	newBalance := acc.Balance + depositedAmount
	if err := db.UpdateAccountBalance(acc.ID, newBalance); err != nil {
		auth.RespondWithError(w, http.StatusInternalServerError, "Failed to update account balance")
		return
	}

	// Create a transaction record
	transaction, err := CreateTransaction(env.DB, &Transaction{
		AccountID:       acc.ID,
		TransactionType: "deposit",
		Amount:          depositedAmount,
		Currency:        req.Currency,
	})
	if err != nil {
		auth.RespondWithError(w, http.StatusInternalServerError, "Failed to create transaction")
		return
	}

	auth.JSON(w, http.StatusOK, transaction)
}

func CreateTransaction(db *sql.DB, transaction *Transaction) (*Transaction, error) {
	query := `INSERT INTO transactions (account_id, transaction_type, amount, currency)
			  VALUES ($1, $2, $3, $4) RETURNING id, timestamp`
	err := db.QueryRow(query, transaction.AccountID, transaction.TransactionType, transaction.Amount, transaction.Currency).Scan(&transaction.ID, &transaction.Timestamp)
	if err != nil {
		return nil, fmt.Errorf("could not create transaction: %w", err)
	}
	return transaction, nil
}
