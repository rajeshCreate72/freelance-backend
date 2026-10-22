package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"freelance-backend/db"
	"freelance-backend/middleware"
	"net/http"
	"strings"
	"unicode/utf8"
)

type CreateReviewRequest struct {
	Rating int `json:"rating"`
	Comment string `json:"comment"`
}

// CreateReview handles POST /jobs/{job_id}/reviews
// The authenticated role determines which participant is reviewing
func CreateReview(conn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reviewerRole, ok := middleware.Role(r)
		if !ok || (reviewerRole != "client" && reviewerRole != "freelancer") {
			http.Error(w, "Only clients or freelancers can submit reviews", http.StatusForbidden)
			return
		}

		reviewerID, ok := middleware.AccountID(r)
		if !ok {
			http.Error(w, "Not authenticated", http.StatusUnauthorized)
			return
		}

		jobID := r.PathValue("job_id")
		if !uuidPattern.MatchString(jobID) {
			http.Error(w, "Invalid job ID", http.StatusBadRequest)
			return
		}

		var req CreateReviewRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid review data JSON", http.StatusBadRequest)
			return
		}

		if req.Rating < 1 || req.Rating > 5 {
			http.Error(w, "Rating must be between 1 and 5", http.StatusBadRequest)
			return
		}

		req.Comment = strings.TrimSpace(req.Comment)
		if utf8.RuneCountInString(req.Comment) > 2000 {
			http.Error(w, "Comment must be 2000 characters or fewer", http.StatusBadRequest)
			return
		}

		review, err := db.CreateReview(
			r.Context(),
			conn,
			jobID,
			reviewerID,
			reviewerRole,
			req.Rating,
			req.Comment,
		)
		if errors.Is(err, db.ErrReviewNotAllowed) {
			http.Error(w, "You cannot review this job or its assignment is not accepted", http.StatusForbidden)
			return
		}
		if errors.Is(err, db.ErrReviewAlreadySubmitted) {
			http.Error(w, "You have already submitted a review for this job", http.StatusConflict)
			return
		}
		if err != nil {
			http.Error(w, "Could not save review", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(review)
	}
}

// ListFreelancerReviews handles
// GET /freelancers/{freelancer_id}/reviews.
func ListFreelancerReviews(conn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		freelancerID := r.PathValue("freelancer_id")
		if !uuidPattern.MatchString(freelancerID) {
			http.Error(w, "Invalid freelancer ID", http.StatusBadRequest)
			return
		}

		reviews, err := db.ListFreelancerReviews(
			r.Context(),
			conn,
			freelancerID,
		)
		if err != nil {
			http.Error(w, "Could not retrieve freelancer reviews", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-type", "application/json")
		_ = json.NewEncoder(w).Encode(reviews)
	}
}

// ListClientReviews handles GET /clients/{client_id}/reviews.
func ListClientReviews(conn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientID := r.PathValue("client_id")
		if !uuidPattern.MatchString(clientID) {
			http.Error(w, "Invalid client ID", http.StatusBadRequest)
			return
		}

		reviews, err := db.ListClientReviews(r.Context(), conn, clientID)
		if err != nil {
			http.Error(w, "Could not retrieve client reviews", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(reviews)
	}
}