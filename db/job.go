package db

import (
	"context"
	"database/sql"
	"time"
)


type Job struct {
	ID          string    `json:"id"`
	ClientID    string    `json:"client_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type Application struct {
	ID           string    `json:"id"`
	JobID        string    `json:"job_id"`
	FreelancerID string    `json:"freelancer_id"`
	CreatedAt    time.Time `json:"created_at"`
}

// JobApplication contains application details shown to the job's owner.
type JobApplication struct {
	ID                 string    `json:"id"`
	JobID              string    `json:"job_id"`
	FreelancerID       string    `json:"freelancer_id"`
	FreelancerUsername string    `json:"freelancer_username"`
	CreatedAt          time.Time `json:"created_at"`
}

type FreelancerApplication struct {
	ID          string    `json:"id"`
	JobID       string    `json:"job_id"`
	JobTitle    string    `json:"job_title"`
	Description string    `json:"description"`
	AppliedAt   time.Time `json:"applied_at"`
}

func CreateJob(
	ctx context.Context,
	conn *sql.DB,
	clientID string,
	title string,
	description string,
) (Job, error) {
	const query = `
		INSERT INTO jobs (client_id, title, description)
		VALUES ($1, $2, $3)
		RETURNING id::text, client_id::text, title, description, created_at
	`

	var job Job

	err := conn.QueryRowContext(
		ctx,
		query, 
		clientID,
		title,
		description,
	).Scan(
		&job.ID,
		&job.ClientID,
		&job.Title,
		&job.Description,
		&job.CreatedAt,
	)
	if err != nil {
		return Job{}, err
	}

	return job, nil
}

// ListJobs retrives all jobs, newest first
func ListJobs(ctx context.Context, conn *sql.DB) ([]Job, error) {
	const query = `
		SELECT id::text, client_id::text, title, description, created_at
		FROM jobs
		ORDER BY created_at DESC
	`

	rows, err := conn.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// start with empty array of jobs returns[] rather than null
	jobs := make([]Job, 0)

	for rows.Next() {
		var job Job

		err := rows.Scan(
			&job.ID,
			&job.ClientID,
			&job.Title,
			&job.Description,
			&job.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		jobs = append(jobs, job)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return jobs, nil
}

// GetJobByID finds one job using its ID
func GetJobByID(
	ctx context.Context,
	conn *sql.DB,
	jobID string,
) (Job, error) {
	const query = `
		SELECT id::text, client_id::text, title, description, created_at
		FROM jobs
		WHERE id = $1
	`

	var job Job

	err := conn.QueryRowContext(ctx, query, jobID).Scan(
		&job.ID,
		&job.ClientID,
		&job.Title,
		&job.Description,
		&job.CreatedAt,
	)
	if err != nil {
		return Job{}, err
	}

	return job, nil
}

func CreateApplication(
	ctx context.Context,
	conn *sql.DB,
	jobID string,
	freelancerID string,
) (Application, error) {
	var exists bool
	err := conn.QueryRowContext(
		ctx,
		`SELECT EXISTS (SELECT 1 FROM jobs WHERE id = $1)`,
		jobID,
	).Scan(&exists)
	if err != nil {
		return Application{}, err
	}
	if !exists {
		return Application{}, ErrJobNotFound
	}

	const query = `
		INSERT INTO applications (job_id, freelancer_id)
		VALUES ($1, $2)
		RETURNING id::text, job_id::text, freelancer_id::text, created_at
	`
	var application Application

	err = conn.QueryRowContext(
		ctx,
		query,
		jobID,
		freelancerID,
	).Scan(
		&application.ID,
		&application.JobID,
		&application.FreelancerID,
		&application.CreatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return Application{}, ErrApplicationAlreadyExists
		}
		return Application{}, err
	}

	return application, nil
}

// ListApplicationsForClientJob returns applicants/freelancers only if the client owns the job.
// It returns ErrJobNotFound when the job doesn't exist or belongs to another client.
func ListApplicationsForClientJob(
	ctx context.Context,
	conn *sql.DB,
	jobID string,
	clientID string,
) ([]JobApplication, error) {
	var ClientOwnsJob bool

	err := conn.QueryRowContext(
		ctx,
		`SELECT EXISTS (
			SELECT 1 FROM jobs WHERE id = $1 AND client_id = $2
		)`,
		jobID,
		clientID,
	).Scan(&ClientOwnsJob)
	if err != nil {
		return nil, err
	}
	if !ClientOwnsJob {
		return nil, ErrJobNotFound
	}

	const query = `
		SELECT a.id::text, a.job_id::text, a.freelancer_id::text,
			f.username, a.created_at
		FROM applications a
		JOIN freelancers f ON f.id = a.freelancer_id
		WHERE a.job_id = $1
		ORDER BY a.created_at DESC
	`

	rows, err := conn.QueryContext(ctx, query, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applications := make([]JobApplication, 0)

	for rows.Next() {
		var application JobApplication

		err := rows.Scan(
			&application.ID,
			&application.JobID,
			&application.FreelancerID,
			&application.FreelancerUsername,
			&application.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		applications = append(applications, application)
	}
	
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return applications, nil
}

// ListApplicationsByFreelancer returns jobs a freelancer applied to
func ListApplicationsByFreelancer(
	ctx context.Context,
	conn *sql.DB,
	freelancerID string,
) ([]FreelancerApplication, error) {
	const query = `
		SELECT a.id::text, a.job_id::text, j.title, j.description, a.created_at
		FROM applications a
		JOIN jobs j ON j.id = a.job_id
		WHERE a.freelancer_id = $1
		ORDER BY a.created_at DESC
	`

	rows, err := conn.QueryContext(ctx, query, freelancerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applications := make([]FreelancerApplication, 0)

	for rows.Next() {
		var application FreelancerApplication

		if err := rows.Scan(
			&application.ID,
			&application.JobID,
			&application.JobTitle,
			&application.Description,
			&application.AppliedAt,
		); err != nil {
			return nil, err
		}

		applications = append(applications, application)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return applications, nil
}

// ListJobsByClient returns the jobs created by one client, newest first.
func ListJobsByClient(
	ctx context.Context,
	conn *sql.DB,
	clientID string,
) ([]Job, error) {
	const query = `
		SELECT id::text, client_id::text, title, description, created_at
		FROM jobs
		WHERE client_id = $1
		ORDER BY created_at DESC
	`

	rows, err := conn.QueryContext(ctx, query, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := make([]Job, 0)

	for rows.Next() {
		var job Job

		if err := rows.Scan(
			&job.ID,
			&job.ClientID,
			&job.Title,
			&job.Description,
			&job.CreatedAt,
		); err != nil {
			return nil, err
		}

		jobs = append(jobs, job)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return jobs, nil
}

