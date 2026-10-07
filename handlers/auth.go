package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"freelance-backend/db"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// LoginRequest represents the JSON body sent to POST /login.
type LoginRequest struct {
	Email string `json:"email"`
	Password string `json:"password"`
}

// LoginClaims defines the account data stored in the signed token.
type LoginClaims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}


// Login verifies credentials and returns a signed, short-lived access token.
func Login(conn *sql.DB, jwtSecret []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req LoginRequest

		// Decode the incoming JSON request body.
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		req.Email = strings.TrimSpace(req.Email)
		if req.Email == "" || req.Password == "" {
			http.Error(w, "Email and password are required", http.StatusBadRequest)
			return
		}
		
		// Load the stored password hash and role for this email.
		account, err := db.FindLoginAccounts(r.Context(), conn, req.Email)
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Invalid email or password", http.StatusUnauthorized)
			return
		} 

		if err != nil {
			log.Println("FindLoginAccounts error:", err)
			http.Error(w, "Failed to login", http.StatusInternalServerError)
    		return
		}

		// Compare the submitted password with the bcrypt hash from the database.
		if err := bcrypt.CompareHashAndPassword(
			[]byte(account.PasswordHash),
			[]byte(req.Password),
		); err != nil {
			http.Error(w, "Invalid email or password", http.StatusUnauthorized)
			return
		}

		expiresAt := time.Now().Add(15 * time.Minute)

		// Put the authenticated account ID and role into the signed token.
		claims := LoginClaims{
			Role: account.Role,
			RegisteredClaims: jwt.RegisteredClaims{
				Subject: account.ID,
				IssuedAt: jwt.NewNumericDate(time.Now()),
				ExpiresAt: jwt.NewNumericDate(expiresAt),
			},
		}

		// Sign the token with the server-only JWT secret.
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString(jwtSecret)
		if err != nil {
			http.Error(w, "Failed to create login token", http.StatusInternalServerError)
			return
		}

		// Return the token the client sends in later protected requests.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		json.NewEncoder(w).Encode(map[string]any{
			"access_token": tokenString,
			"token_type": "Bearer",
			"expires_in": int(time.Until(expiresAt).Seconds()),
		})
	} 
}
