package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"freelance-backend/db"
	"freelance-backend/middleware"
	"net/http"
	"net/mail"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

type ClientRegistrationRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func RegisterClient(conn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var req ClientRegistrationRequest

		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Checking required fields
		req.Username = strings.TrimSpace(req.Username)
		req.Email = strings.TrimSpace(req.Email)

		if req.Username == "" || req.Email == "" || req.Password == "" {
			http.Error(w, "All fields are required", http.StatusBadRequest)
			return
		}

		// Validate email format
		parsedEmail, err := mail.ParseAddress(req.Email)
		if err != nil || parsedEmail.Address != req.Email {
			http.Error(w, "Invalid email address", http.StatusBadRequest)
			return
		}

		// Basic password length reqirement
		if len([]rune(req.Password)) < 8 {
			http.Error(w, "Password must be at least 8 characters", http.StatusBadRequest)
			return
		}

		// Hash password using bcrypt, before storing it.
		passwordHash, err := bcrypt.GenerateFromPassword(
			[]byte(req.Password),
			bcrypt.DefaultCost,
		)
		if err != nil {
			http.Error(w, "Failed to process password, please try again.", http.StatusInternalServerError)
			return
		}

		// Creating client in the database
		err = db.CreateClient(
			conn,
			req.Username,
			req.Email,
			string(passwordHash),
		)
		if err != nil {
			if errors.Is(err, db.ErrEmailAlreadyRegistered) {
				http.Error(w, "Email already registered", http.StatusConflict)
				return
			} 

			http.Error(w, "Failed to create client", http.StatusInternalServerError)
			return
		}

		// Respond with success
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)

		json.NewEncoder(w).Encode(map[string]string{
			"message": "Client data created successfully",
		})
	}
}

// GetClientMe returns the profile belonging to the authenticated client.
func GetClientMe(conn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// The middleware from main already verifies JWT
		role, ok := middleware.Role(r)
		if !ok || role != "client" {
			http.Error(w, "Clients only", http.StatusForbidden)
			return
		}

		// Get the client ID from the verified token context.
		clientID, ok := middleware.AccountID(r)
		if !ok {
			http.Error(w, "Not authenticated", http.StatusUnauthorized)
			return
		}

		// Find this client's profile in PostgreSQL.
		profile, err := db.GetClientByID(r.Context(), conn, clientID)
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Client profile not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "Failed to retrive client profile", http.StatusInternalServerError) 
			return
		}

		// Return the profile as JSON
		w.Header().Set("Content-type", "application/json")
		if err := json.NewEncoder(w).Encode(profile); err != nil {
			return
		}

	}
}