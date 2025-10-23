package transactions

import "time"

// --- Models ---

type Transaction struct {
	ID              int       `json:"id"`
	AccountID       string    `json:"account_id"`
	TransactionType string    `json:"transaction_type"`
	Amount          float64   `json:"amount"`
	Currency        string    `json:"currency"`
	Timestamp       time.Time `json:"timestamp"`
}
