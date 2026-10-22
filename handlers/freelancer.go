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

type FreelancerRegistrationRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func RegisterFreelancer(conn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var req FreelancerRegistrationRequest

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

		// Creating freelancer in the database
		err = db.CreateFreelancer(
			conn,
			req.Username,
			req.Email,
			string(passwordHash),
		)

		if err != nil {
			http.Error(w, "Email already registered or failed to create freelancer", http.StatusConflict)
			return
		}

		// Respond with success
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)

		json.NewEncoder(w).Encode(map[string]string{
			"message": "Freelancer data created successfully",
		})
	}
}

// GetFreelancerMe	returns the profile belonging to the authenticated freelancer.
func GetFreelancerMe(conn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// The middleware has verified the JWT token. 
		role, ok := middleware.Role(r)
		if !ok || role != "freelancer" {
			http.Error(w, "Freelancers only", http.StatusForbidden)
			return
		}

		// Get the freelance id from the verified token.
		freelanceID, ok := middleware.AccountID(r)
		if !ok {
			http.Error(w, "Not authenticated", http.StatusUnauthorized)
			return
		}

		profile, err := db.GetFreelanceByID(
			r.Context(),
			conn,
			freelanceID,
		)
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Freelancer profile not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(
				w,
				"Failed to retrive freelancer profile",
				http.StatusInternalServerError,
			)
			return 
		}

		// Encode writes the profile as JSON to the HTTP response
		w.Header().Set("Content-type", "application/json")
		if err := json.NewEncoder(w).Encode(profile); err != nil {
			return
		}
	}
}