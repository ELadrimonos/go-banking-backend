package auth

import (
	"crypto/rand"
	"errors"
	"fmt"
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

func GetUserIDFromContext(r *http.Request) (string, error) {
	userID, ok := r.Context().Value("userID").(string)
	if !ok || userID == "" {
		return "", errors.New("unauthorized")
	}
	return userID, nil
}

func GeneratePINAndHash() (string, string, error) {
	pin, err := generateInitialPIN(6)
	if err != nil {
		return "", "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pin), bcrypt.DefaultCost)
	if err != nil {
		return "", "", err
	}
	return pin, string(hash), nil
}

func generateInitialPIN(n int) (string, error) {
	const digits = "0123456789"
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("could not generate random bytes: %w", err)
	}
	for i := range b {
		b[i] = digits[int(b[i])%len(digits)]
	}
	return string(b), nil
}
