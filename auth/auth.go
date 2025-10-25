package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

// --- Models ---

type User struct {
	ID               string    `json:"id"`
	DNI              string    `json:"dni"`
	GeneratedPinHash string    `json:"-"`
	FullName         string    `json:"full_name"`
	Email            string    `json:"email"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type SignupRequest struct {
	FullName string `json:"full_name"`
	DNI      string `json:"dni"`
	Email    string `json:"email"`
}

type LoginRequest struct {
	DNI string `json:"dni"`
	Pin string `json:"pin"`
}

type ChangePasswordRequest struct {
	OldPin string `json:"old_pin"`
	NewPin string `json:"new_pin"`
}

type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// --- Database ---

type DB struct {
	*sql.DB
}

func (db *DB) CreateUser(ctx context.Context, user *User, pinHash string) (string, error) {
	var id string
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("could not begin transaction: %w", err)
	}
	defer func(tx *sql.Tx) {
		_ = tx.Rollback()
	}(tx) // Rollback in case of an error

	query := `INSERT INTO users (dni, generated_pin_hash, full_name, email)
			  VALUES ($1, $2, $3, $4) RETURNING id`
	err = tx.QueryRowContext(ctx, query, user.DNI, pinHash, user.FullName, user.Email).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("could not create user: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("could not commit transaction: %w", err)
	}

	return id, nil
}

func (db *DB) GetUserByDNI(dni string) (*User, error) {
	user := &User{}
	query := `SELECT id, dni, generated_pin_hash, full_name, email, updated_at FROM users WHERE dni = $1`
	err := db.QueryRow(query, dni).Scan(&user.ID, &user.DNI, &user.GeneratedPinHash, &user.FullName, &user.Email, &user.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("could not get user by dni: %w", err)
	}
	return user, nil
}

func (db *DB) GetUserByID(id string) (*User, error) {
	user := &User{}
	query := `SELECT id, dni, generated_pin_hash, full_name, email, updated_at FROM users WHERE id = $1`
	err := db.QueryRow(query, id).Scan(&user.ID, &user.DNI, &user.GeneratedPinHash, &user.FullName, &user.Email, &user.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("could not get user by id: %w", err)
	}
	return user, nil
}

func (db *DB) UpdatePinHash(userID, newPinHash string) error {
	query := `UPDATE users SET generated_pin_hash = $1, updated_at = NOW() WHERE id = $2`
	_, err := db.Exec(query, newPinHash, userID)
	if err != nil {
		return fmt.Errorf("could not update pin hash: %w", err)
	}
	return nil
}

// --- JWT ---

func getJWTKey() []byte {
	key := os.Getenv("JWT_SECRET")
	if key == "" {
		return []byte("my_secret_key") // Default key for development
	}
	return []byte(key)
}

func GenerateTokens(userID string) (string, string, error) {
	// Generate access token
	accessTokenClaims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)), // Access token expires in 15 minutes
		},
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessTokenClaims)
	accessTokenString, err := accessToken.SignedString(getJWTKey())
	if err != nil {
		return "", "", err
	}

	// Generate refresh token
	refreshTokenClaims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)), // Refresh token expires in 7 days
		},
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshTokenClaims)
	refreshTokenString, err := refreshToken.SignedString(getJWTKey())
	if err != nil {
		return "", "", err
	}

	return accessTokenString, refreshTokenString, nil
}

func ValidateJWT(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return getJWTKey(), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}

// --- Handlers ---

type Env struct {
	DB *sql.DB
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (env *Env) SignupHandler(w http.ResponseWriter, r *http.Request) {
	req, ok := r.Context().Value(signupRequestKey).(SignupRequest)
	if !ok {
		RespondWithError(w, http.StatusInternalServerError, "Could not get signup request from context")
		return
	}

	pin, pinHash, err := GeneratePINAndHash()
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to generate PIN")
		return
	}

	db := &DB{env.DB}
	user := &User{DNI: req.DNI, FullName: req.FullName, Email: req.Email}
	userID, err := db.CreateUser(r.Context(), user, pinHash)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	JSON(w, http.StatusCreated, map[string]string{"user_id": userID, "pin": pin})
}

func (env *Env) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	db := &DB{env.DB}
	user, err := db.GetUserByDNI(req.DNI)
	if err != nil || user == nil {
		RespondWithError(w, http.StatusUnauthorized, "Invalid DNI or PIN")
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.GeneratedPinHash), []byte(req.Pin))
	if err != nil {
		RespondWithError(w, http.StatusUnauthorized, "Invalid DNI or PIN")
		return
	}

	accessToken, refreshToken, err := GenerateTokens(user.ID)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to generate tokens")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
	if err != nil {
		return
	}
}

func (env *Env) RefreshHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	claims, err := ValidateJWT(req.RefreshToken)
	if err != nil {
		RespondWithError(w, http.StatusUnauthorized, "Invalid refresh token")
		return
	}

	accessToken, refreshToken, err := GenerateTokens(claims.UserID)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to generate tokens")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
	if err != nil {
		return
	}
}

func (env *Env) StatusHandler(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (env *Env) ChangePasswordHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := GetUserIDFromContext(r)
	if err != nil {
		RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	pin, pinHash, err := GeneratePINAndHash()
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to generate new PIN")
		return
	}

	db := &DB{env.DB}
	user, err := db.GetUserByID(userID)
	if err != nil || user == nil {
		RespondWithError(w, http.StatusNotFound, "User not found")
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.GeneratedPinHash), []byte(req.OldPin))
	if err != nil {
		RespondWithError(w, http.StatusUnauthorized, "Invalid old PIN")
		return
	}

	err = db.UpdatePinHash(userID, pinHash)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to update PIN")
		return
	}

	JSON(w, http.StatusOK, map[string]string{"message": "PIN updated successfully", "new_pin": pin})
}

// --- Middleware ---

func AuthenticationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			RespondWithError(w, http.StatusUnauthorized, "Authorization header required")
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			RespondWithError(w, http.StatusUnauthorized, "Invalid token format")
			return
		}

		claims, err := ValidateJWT(tokenString)
		if err != nil {
			RespondWithError(w, http.StatusUnauthorized, "Invalid token")
			return
		}

		ctx := context.WithValue(r.Context(), "userID", claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
