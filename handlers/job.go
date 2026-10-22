package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"freelance-backend/db"
	"freelance-backend/middleware"
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"
)

type CreateJobRequest struct {
	Title string `json:"title"`
	Description string `json:"description"`
}

var uuidPattern = regexp.MustCompile(
	`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`,
)

// CreateJob handles POST /jobs.
func CreateJob(conn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// The middleware has verified the JWT. Only clients may post jobs.
		role, ok := middleware.Role(r)
		if !ok || role != "client" {
			http.Error(w, "Clients only", http.StatusForbidden)
			return
		}

		// Read the verified client's ID from the token context.
		clientID, ok := middleware.AccountID(r)
		if !ok {
			http.Error(w, "Not authenticated", http.StatusUnauthorized)
			return
		}

		// Decode the JSON request body into the request struct.
		var req CreateJobRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Remove whitespace around the supplied text.
		req.Title = strings.TrimSpace(req.Title)
		req.Description = strings.TrimSpace(req.Description)

		// Both fields are required.
		if req.Title == "" || req.Description == "" {
			http.Error(w, "Title and description are required", http.StatusBadRequest)
			return
		}

		// The database column allows at most 200 title characters.
		if utf8.RuneCountInString(req.Title) > 200 {
			http.Error(w, "Title must be 200 characters or fewer", http.StatusBadRequest)
			return
		}

		// Save the job linked to the authenticated client.
		job, err := db.CreateJob(
			r.Context(),
			conn,
			clientID,
			req.Title,
			req.Description,
		)
		if err != nil {
			http.Error(w, "Failed to create job", http.StatusInternalServerError)
			return
		}

		// Return the new job as JSON with HTTP 201 Created.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(job)
	}
}

// ListJobs handles GET /jobs
func ListJobs(conn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jobs, err := db.ListJobs(r.Context(), conn)
		if err != nil {
			http.Error(w, "Failed to retrive jobs", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-type", "application/json")
		if err := json.NewEncoder(w).Encode(jobs); err != nil {
			return
		}
	}
}

// GetJob handles GET /jobs/{id}.
func GetJob(conn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// "id" comes from {id} part of the registered route
		jobID := r.PathValue("id")

		job, err := db.GetJobByID(r.Context(), conn, jobID)
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Job not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "Failed to retrive job", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-type", "application/json")
		if err := json.NewEncoder(w).Encode(job); err != nil {
			return
		}
	}
}

// ApplyToJob handles POST /jobs/{job_id}/applications
func ApplyToJob(conn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only logged-in freelancer may apply
		role, ok := middleware.Role(r)
		if !ok || role != "freelancer" {
			http.Error(w, "Only freelancer can apply for the job", http.StatusForbidden)
			return
		}

		freelancerID, ok := middleware.AccountID(r)
		if !ok {
			http.Error(w, "Not authenticated", http.StatusUnauthorized)
			return
		}

		jobID := r.PathValue("job_id")
		if !uuidPattern.MatchString(jobID) {
			http.Error(w, "Invalid job ID", http.StatusBadRequest)
			return
		}

		application, err := db.CreateApplication(
			r.Context(),
			conn,
			jobID,
			freelancerID,
		)
		if errors.Is(err, db.ErrJobNotFound) {
			http.Error(w, "Job  not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, db.ErrApplicationAlreadyExists) {
			http.Error(w, "You have already applied to this job", http.StatusConflict)
			return
		}
		if err != nil {
			http.Error(w, "Failed to apply to job", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-type", "applicaton/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(application)
	}
}

// ListJobApplications handles GET /jobs/{job_id}/applications.
func ListJobApplications(conn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only a signed-in client can view applicants.
		role, ok := middleware.Role(r)
		if !ok || role != "client" {
			http.Error(w, "Only clients can view applications", http.StatusForbidden)
			return
		}

		clientID, ok := middleware.AccountID(r)
		if !ok {
			http.Error(w, "Not authenticated", http.StatusUnauthorized)
			return
		}

		// This ID comes from the {job_id} part of the route.
		jobID := r.PathValue("job_id")
		if !uuidPattern.MatchString(jobID) {
			http.Error(w, "Invalid job ID", http.StatusBadRequest)
			return
		}

		applications, err := db.ListApplicationsForClientJob(
			r.Context(),
			conn,
			jobID,
			clientID,
		)
		if errors.Is(err, db.ErrJobNotFound) {
			// Also used when another client owns the job,
			// So, its existence not disclosed
			http.Error(w, "Job not found", http.StatusNotFound)
			return
		}

		if err != nil {
			http.Error(w, "Failed to retrive application", http.StatusInternalServerError)
			return
		}
		
		w.Header().Set("Content-type", "application/json")
		if err := json.NewEncoder(w).Encode(applications); err != nil {
			return
		}
	}
}

// ListApplicationsByFreelancer returns the jobs a freelancer applied to.
func ListMyApplications(conn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		role, ok := middleware.Role(r)
		if !ok || role != "freelancer" {
			http.Error(w, "Only freelancer can access the data", http.StatusForbidden)
			return
		}

		// The freelancer ID comes from their verified JWT
		freelancerID, ok := middleware.AccountID(r)
		if !ok {
			http.Error(w, "Not authenticated", http.StatusUnauthorized)
			return
		}

		applications, err := db.ListApplicationsByFreelancer(
			r.Context(),
			conn,
			freelancerID,
		)
		if err != nil {
			http.Error(w, "Failed to retrive applications", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-type", "application/json")
		if err := json.NewEncoder(w).Encode(applications); err != nil {
			return
		}
	}
}

// ListMyJobs handles GET /clients/me/jobs.
func ListMyJobs(conn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		role, ok := middleware.Role(r)
		if !ok || role != "client" {
			http.Error(w, "Client only", http.StatusForbidden)
			return 
		}

		// This ID comes from the authenticated client's JWT.
		clientID, ok := middleware.AccountID(r)
		if !ok {
			http.Error(w, "Not authenticated", http.StatusUnauthorized)
			return
		}

		jobs, err := db.ListJobsByClient(r.Context(), conn, clientID)
		if err != nil {
			http.Error(w, "Failed to retrive your jobs", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-type", "application/json")
		if err := json.NewEncoder(w).Encode(jobs); err != nil {
			return
		}
	}
}