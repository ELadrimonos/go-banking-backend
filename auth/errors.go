package auth

import (
	"encoding/json"
	"net/http"
)

// --- Error Handling ---

type ErrorResponse struct {
	Error string `json:"error"`
}

func RespondWithError(w http.ResponseWriter, code int, message string) {
	response := ErrorResponse{Error: message}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		return
	}
}
