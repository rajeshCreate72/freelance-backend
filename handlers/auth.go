package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
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

// RefreshRequest has Refresh token
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
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

		accessToken, accessExpiresAt, err := createAccessToken(
			account.ID,
			account.Role,
			jwtSecret,
		)
		if err != nil {
			log.Println("Create access token error:", err)
			http.Error(w, "Failed to create login token", http.StatusInternalServerError)
			return
		}

		refreshToken, refreshTokenHash, err := createRefreshToken()
		if err != nil {
			log.Println("Create refresh token error:", err)
			http.Error(w, "Failed to create refresh token", http.StatusInternalServerError)
			return
		}

		refreshExpiresAt := time.Now().Add(30 * 24 * time.Hour)

		// Store the hash in the database. The raw token is returned to the client.
		if err := db.SaveRefreshToken(
			r.Context(),
			conn,
			refreshTokenHash,
			account.ID,
			account.Role,
			refreshExpiresAt,
		); err != nil {
			log.Printf("SaveRefreshToken failed: %v", err)
			http.Error(w, "Failed to create login session", http.StatusInternalServerError)
			return
		}

		// Return both tokens. The client uses the access token for protected
		// requests and sends the refresh token to POST /refresh when needed.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(map[string]any{
			"access_token":  accessToken,
			"token_type":    "Bearer",
			"expires_in":    int(time.Until(accessExpiresAt).Seconds()),
			"refresh_token": refreshToken,
		}); err != nil {
			log.Println("Write login response error:", err)
		}
	} 
}

// Refresh will create a access token from refresh_token fetched from user
func Refresh(conn *sql.DB, jwtSecrect []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req RefreshRequest

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON request", http.StatusBadRequest)
			return
		}

		if strings.TrimSpace(req.RefreshToken) == "" {
			http.Error(w, "refresh_token is required in the request", http.StatusBadRequest)
			return
		}

		newRefreshToken, newRefreshHash, err := createRefreshToken()

		if err != nil {
			http.Error(w, "Failed to create refresh token", http.StatusInternalServerError)
			return
		}

		oldHashBytes := sha256.Sum256([]byte(req.RefreshToken))
		oldHash := hex.EncodeToString(oldHashBytes[:])

		newRefreshExpiry := time.Now().Add(30 * 24 * time.Hour)

		accountID, role, err := db.RotateRefreshToken(
			r.Context(),
			conn,
			oldHash,
			newRefreshHash,
			newRefreshExpiry,
		)
		if errors.Is(err, db.ErrInvalidRefreshToken) {
			http.Error(w, "Invalid or expired refresh token", http.StatusUnauthorized)
			return
		}
		if err != nil {
			log.Printf("RotateRefreshToken failed: %v", err)
			http.Error(w, "Failed to refresh session", http.StatusInternalServerError)
			return
		}

		accessToken, accessExpiresAt, err := createAccessToken(
			accountID,
			role,
			jwtSecrect,
		)
		if err != nil {
			http.Error(w, "Failed to create access token", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": accessToken,
			"token_type": "Bearer",
			"expires_in": int(time.Until(accessExpiresAt).Seconds()),
			"refresh_token": newRefreshToken,
		})
	}
}


// createAccessToken signs a short-lived JWT for protected API requests.
func createAccessToken(
	accountID string,
	role string,
	jwtSecrect []byte,
) (string, time.Time, error) {
	if len(jwtSecrect) == 0 {
		return "", time.Time{}, errors.New("JWT_SECRET is not configured")
	}

	now := time.Now()
	expiresAt := now.Add(1 * time.Minute)

	claims := LoginClaims{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: accountID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(jwtSecrect)
	return signed, expiresAt, err
}

// createRefreshToken makes a strong random token and the hash to store in DB.
// crpto/rand supplies cryptographically secure random bytes.
func createRefreshToken() (plainToken string, tokenHash string, err error) {
	randomBytes	:= make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", "", err
	}

	plainToken = base64.RawURLEncoding.EncodeToString(randomBytes)
	hash := sha256.Sum256([]byte(plainToken))
	tokenHash = hex.EncodeToString(hash[:])

	return plainToken, tokenHash, nil
}