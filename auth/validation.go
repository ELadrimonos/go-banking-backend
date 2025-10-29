package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"

	"github.com/ELadrimonos/national-document-validator/validators"
	"github.com/ELadrimonos/national-document-validator/validators/es"
)

// --- Context Keys ---

type contextKey string

const signupRequestKey contextKey = "signupRequest"

// --- Validation Middleware ---

func ValidateSignupRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req SignupRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			RespondWithError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if err := validateSignupData(req); err != nil {
			RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}

		// If validation is successful, store the request in the context and call the next handler
		ctx := context.WithValue(r.Context(), signupRequestKey, req)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func validateSignupData(req SignupRequest) error {
	if err := validateDNI(req.DNI); err != nil {
		return err
	}
	if err := validateFullName(req.FullName); err != nil {
		return err
	}
	if err := validateEmail(req.Email); err != nil {
		return err
	}
	return nil
}

// TODO Get user country
func validateDNI(dniValue string) error {
	validator := validators.NewValidator()
	validator.Register("es", &es.ESValidator{})

	err := validator.Validate("es", dniValue)
	if err != nil {
		return fmt.Errorf("invalid DNI: %w", err)
	}

	return nil
}

func validateFullName(fullName string) error {
	if len(fullName) < 3 {
		return fmt.Errorf("full name must be at least 3 characters long")
	}
	return nil
}

func validateEmail(email string) error {
	// A simple regex for email validation
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !re.MatchString(email) {
		return fmt.Errorf("invalid email format")
	}
	return nil
}
