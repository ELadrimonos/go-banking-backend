package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

// --- Models ---

type User struct {
	ID               string `json:"id"`
	DNI              string `json:"dni"`
	GeneratedPinHash string `json:"-"`
	FullName         string `json:"full_name"`
	Email            string `json:"email"`
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

type Claims struct {
	UserID string `json:"user_id"`
	jwt.StandardClaims
}

// --- Database ---

type DB struct {
	*sql.DB
}

func (db *DB) CreateUser(user *User, pinHash string) (string, error) {
	var id string
	query := `INSERT INTO users (dni, generated_pin_hash, full_name, email)
			  VALUES ($1, $2, $3, $4) RETURNING id`
	err := db.QueryRow(query, user.DNI, pinHash, user.FullName, user.Email).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("could not create user: %w", err)
	}
	return id, nil
}

func (db *DB) GetUserByDNI(dni string) (*User, error) {
	user := &User{}
	query := `SELECT id, dni, generated_pin_hash, full_name, email FROM users WHERE dni = $1`
	err := db.QueryRow(query, dni).Scan(&user.ID, &user.DNI, &user.GeneratedPinHash, &user.FullName, &user.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("could not get user by dni: %w", err)
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

var jwtKey = []byte("my_secret_key") // In production, use a secure, configured key

func GenerateJWT(userID string) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		UserID: userID,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

func ValidateJWT(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
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

func (env *Env) SignupHandler(w http.ResponseWriter, r *http.Request) {
	var req SignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	pin, err := generateRandomPIN(6)
	if err != nil {
		http.Error(w, "Failed to generate PIN", http.StatusInternalServerError)
		return
	}

	pinHash, err := bcrypt.GenerateFromPassword([]byte(pin), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Failed to hash PIN", http.StatusInternalServerError)
		return
	}

	db := &DB{env.DB}
	user := &User{DNI: req.DNI, FullName: req.FullName, Email: req.Email}
	userID, err := db.CreateUser(user, string(pinHash))
	if err != nil {
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"user_id": userID, "pin": pin})
}

func (env *Env) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	db := &DB{env.DB}
	user, err := db.GetUserByDNI(req.DNI)
	if err != nil || user == nil {
		http.Error(w, "Invalid DNI or PIN", http.StatusUnauthorized)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.GeneratedPinHash), []byte(req.Pin))
	if err != nil {
		http.Error(w, "Invalid DNI or PIN", http.StatusUnauthorized)
		return
	}

	tokenString, err := GenerateJWT(user.ID)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": tokenString})
}

func (env *Env) ChangePasswordHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(string)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	newPin, err := generateRandomPIN(6)
	if err != nil {
		http.Error(w, "Failed to generate new PIN", http.StatusInternalServerError)
		return
	}

	newPinHash, err := bcrypt.GenerateFromPassword([]byte(newPin), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Failed to hash new PIN", http.StatusInternalServerError)
		return
	}

	db := &DB{env.DB}
	err = db.UpdatePinHash(userID, string(newPinHash))
	if err != nil {
		http.Error(w, "Failed to update PIN", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"pin": newPin})
}

// --- Middleware ---

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			http.Error(w, "Invalid token format", http.StatusUnauthorized)
			return
		}

		claims, err := ValidateJWT(tokenString)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), "userID", claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// --- Helpers ---

func generateRandomPIN(n int) (string, error) {
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
