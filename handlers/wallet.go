package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"freelance-backend/db"
	"freelance-backend/middleware"
	"net/http"
	"strings"
	"time"

	solanaSDK "github.com/gagliardetto/solana-go"
)

type WalletNonceRequest struct {
	WalletAddress string `json:"wallet_address"`
}

// CreateWalletNonce creates a five-minute challenge for a signed-in account.
func CreateWalletNonce(conn *sql.DB) http.HandlerFunc {
	return func (w http.ResponseWriter, r *http.Request)  {
		accountID, ok := middleware.AccountID(r)
		if !ok {
			http.Error(w, "Not authenticated", http.StatusUnauthorized)
			return
		}

		role, ok := middleware.Role(r)
		if !ok {
			http.Error(w, "Invalid account role", http.StatusUnauthorized)
			return
		}

		var req WalletNonceRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		req.WalletAddress = strings.TrimSpace(req.WalletAddress)
		if _, err := solanaSDK.PublicKeyFromBase58(req.WalletAddress); err != nil {
			http.Error(w, "Invalid Solana wallet address", http.StatusBadRequest)
			return
		}

		// 32 random bytes make the nonce hard to guess.
		randomBytes := make([]byte, 32)
		if _, err := rand.Read(randomBytes); err != nil {
			http.Error(w, "Could not create wallet nonce", http.StatusInternalServerError)
			return
		}
		nonce := base64.RawURLEncoding.EncodeToString(randomBytes)

		expiresAt := time.Now().UTC().Add(5 * time.Minute)
		if err := db.CreateWalletNonce(
			r.Context(),
			conn,
			accountID,
			role,
			req.WalletAddress,
			nonce,
			expiresAt,
		); err != nil {
			http.Error(w, "Could not save wallet nonce", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any {
			"wallet_address": req.WalletAddress,
			"nonce": nonce,
			"expires_at": expiresAt.UTC().Format(time.RFC3339),
			"message": fmt.Sprintf(
				"Sign this nonce to prove you control wallet %s: %s",
				req.WalletAddress,
				nonce,
			),
		})
	}
}