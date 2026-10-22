package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"freelance-backend/db"
	"freelance-backend/middleware"
	"net/http"
	"strings"
)

type AssginFreelancerRequest struct {
	// This is the application ID returned by GET /jobs/{job_id}/applications.
	ApplicationID string `json:"application_id"`
}

// AssignFreelancer handles POST /jobs/{job_id}/assignments.
// Only the client who owns the job can select one of its applicants.
func AssignFreelancer(conn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		role, ok := middleware.Role(r)
		if !ok || role != "client" {
			http.Error(w, "Only clients can assign a freelancer", http.StatusForbidden)
			return
		}

		jobID := r.PathValue("job_id")
		if !uuidPattern.MatchString(jobID) {
			http.Error(w, "Invalid job ID", http.StatusBadRequest)
			return
		}

		clientID, ok := middleware.AccountID(r)
		if !ok {
			http.Error(w, "Not authenticated", http.StatusUnauthorized)
			return
		}

		var req AssginFreelancerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invaid JSON", http.StatusBadRequest)
			return
		}

		req.ApplicationID = strings.TrimSpace(req.ApplicationID)
		if !uuidPattern.MatchString(req.ApplicationID) {
			http.Error(w, "A valid application_id is required", http.StatusBadRequest)
		}

		assignment, err := db.AssignFreelancer(
			r.Context(),
			conn,
			jobID,
			clientID,
			req.ApplicationID,
		)
		if errors.Is(err, db.ErrJobApplicationNotFound) {
			http.Error(w, "Job or application not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, db.ErrJobAlreadyAssigned) {
			http.Error(w, "A freelancer has already been selected for this job", http.StatusConflict)
			return
		}
		if err != nil {
			http.Error(w, "Could not assign freelancer", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(assignment)

	}
}

// AcceptAssignment handles POST /jobs/{job_id}/assignment/accept
// Only the selected freelancer can be able to accept.
func AcceptAssignment(conn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		role, ok := middleware.Role(r)
		if !ok || role != "freelancer" {
			http.Error(w, "Only assigned freelancers can accept the job", http.StatusForbidden)
			return
		}

		freelancerID, ok := middleware.AccountID(r)
		if !ok {
			http.Error(w, "Not authenticated", http.StatusUnauthorized)
			return
		}

		jobID := r.PathValue("job_id")
		if !uuidPattern.MatchString(jobID) {
			http.Error(w, "Invalid Job id passed through endpoint", http.StatusBadRequest)
			return
		}

		assingment, err := db.AcceptAssignment(
			r.Context(),
			conn,
			jobID,
			freelancerID,
		)
		if errors.Is(err, db.ErrAssignmentNotPending) {
			http.Error(w, "No pending assignment for this freelancer and job", http.StatusConflict)
			return
		}
		if err != nil {
			http.Error(w, "Could not accept assignment", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-type", "application/json")
		_ = json.NewEncoder(w).Encode(assingment)
	}
}