package db

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"
)

type Job struct {
	ID                   string    `json:"id"`
	SolanaJobID          string    `json:"solana_job_id"`
	ClientID             string    `json:"client_id"`
	Title                string    `json:"title"`
	Description          string    `json:"description"`
	BudgetLamports       string    `json:"budget_lamports"`
	FundedAmountLamports string    `json:"funded_amount_lamports"`
	Funded               bool      `json:"funded"`
	CreatedAt            time.Time `json:"created_at"`
}

type Application struct {
	ID           string    `json:"id"`
	JobID        string    `json:"job_id"`
	FreelancerID string    `json:"freelancer_id"`
	CreatedAt    time.Time `json:"created_at"`
}

// JobApplication contains applicant details shown to the job's owner.
type JobApplication struct {
	ID                 string    `json:"id"`
	JobID              string    `json:"job_id"`
	FreelancerID       string    `json:"freelancer_id"`
	FreelancerUsername string    `json:"freelancer_username"`
	CreatedAt          time.Time `json:"created_at"`
}

type FreelancerApplication struct {
	ID             string    `json:"id"`
	JobID          string    `json:"job_id"`
	JobTitle       string    `json:"job_title"`
	Description    string    `json:"description"`
	BudgetLamports string    `json:"budget_lamports"`
	Funded         bool      `json:"funded"`
	AppliedAt      time.Time `json:"applied_at"`
}

func CreateJob(
	ctx context.Context,
	conn *sql.DB,
	clientID string,
	title string,
	description string,
	budgetLamports int64,
) (Job, error) {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return Job{}, err
	}
	defer tx.Rollback()

	const insertJob = `
		INSERT INTO jobs (
			client_id,
			title,
			description,
			budget_lamports
		)
		VALUES ($1, $2, $3, $4)
		RETURNING
			id::text,
			solana_job_id::text,
			client_id::text,
			title,
			description,
			budget_lamports::text,
			created_at
	`



	var job Job
	err = tx.QueryRowContext(
		ctx,
		insertJob,
		clientID,
		title,
		description,
		budgetLamports,
	).Scan(
		&job.ID,
		&job.SolanaJobID,
		&job.ClientID,
		&job.Title,
		&job.Description,
		&job.BudgetLamports,
		&job.CreatedAt,
	)
	if err != nil {
		return Job{}, err
	}

	// Create the budget row now. NULL means escrow funds have not arrived yet.
	const insertEscrowBudget = `
		INSERT INTO job_escrows (
			job_id,
			budget_lamports,
			funded_amount_lamports
		)
		VALUES ($1, $2, NULL)
	`

	_, err = tx.ExecContext(
		ctx,
		insertEscrowBudget,
		job.ID,
		budgetLamports,
	)
	if err != nil {
		return Job{}, err
	}

	if err := tx.Commit(); err != nil {
		return Job{}, err
	}

	job.BudgetLamports = strconv.FormatInt(budgetLamports, 10)
	job.FundedAmountLamports = ""
	job.Funded = false

	return job, nil
}

// ListJobs returns all jobs, newest first.
func ListJobs(ctx context.Context, conn *sql.DB) ([]Job, error) {
	const query = `
		SELECT
			j.id::text,
			j.solana_job_id::text,
			j.client_id::text,
			j.title,
			j.description,
			j.budget_lamports::text,
			COALESCE(e.funded_amount_lamports::text, ''),
			(e.job_id IS NOT NULL),
			j.created_at
		FROM jobs j
		LEFT JOIN job_escrows e ON e.job_id = j.id
		ORDER BY j.created_at DESC
	`

	rows, err := conn.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := make([]Job, 0)

	for rows.Next() {
		var job Job

		if err := rows.Scan(
			&job.ID,
			&job.SolanaJobID,
			&job.ClientID,
			&job.Title,
			&job.Description,
			&job.BudgetLamports,
			&job.FundedAmountLamports,
			&job.Funded,
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

// GetJobByID finds one job using its backend UUID.
func GetJobByID(
	ctx context.Context,
	conn *sql.DB,
	jobID string,
) (Job, error) {
	const query = `
		SELECT
			j.id::text,
			j.solana_job_id::text,
			j.client_id::text,
			j.title,
			j.description,
			j.budget_lamports::text,
			COALESCE(e.funded_amount_lamports::text, ''),
			(e.job_id IS NOT NULL),
			j.created_at
		FROM jobs j
		LEFT JOIN job_escrows e ON e.job_id = j.id
		WHERE j.id = $1
	`

	var job Job
	err := conn.QueryRowContext(ctx, query, jobID).Scan(
		&job.ID,
		&job.SolanaJobID,
		&job.ClientID,
		&job.Title,
		&job.Description,
		&job.BudgetLamports,
		&job.FundedAmountLamports,
		&job.Funded,
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

// ListApplicationsForClientJob returns applicants only if the client owns the job.
// It returns ErrJobNotFound when the job doesn't exist or belongs to another client.
func ListApplicationsForClientJob(
	ctx context.Context,
	conn *sql.DB,
	jobID string,
	clientID string,
) ([]JobApplication, error) {
	var clientOwnsJob bool

	err := conn.QueryRowContext(
		ctx,
		`SELECT EXISTS (
			SELECT 1 FROM jobs WHERE id = $1 AND client_id = $2
		)`,
		jobID,
		clientID,
	).Scan(&clientOwnsJob)
	if err != nil {
		return nil, err
	}
	if !clientOwnsJob {
		return nil, ErrJobNotFound
	}

	const query = `
		SELECT
			a.id::text,
			a.job_id::text,
			a.freelancer_id::text,
			f.username,
			a.created_at
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

		if err := rows.Scan(
			&application.ID,
			&application.JobID,
			&application.FreelancerID,
			&application.FreelancerUsername,
			&application.CreatedAt,
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

// ListApplicationsByFreelancer returns jobs a freelancer applied to.
func ListApplicationsByFreelancer(
	ctx context.Context,
	conn *sql.DB,
	freelancerID string,
) ([]FreelancerApplication, error) {
	const query = `
		SELECT
			a.id::text,
			a.job_id::text,
			j.title,
			j.description,
			j.budget_lamports::text,
			(e.job_id IS NOT NULL),
			a.created_at
		FROM applications a
		JOIN jobs j ON j.id = a.job_id
		LEFT JOIN job_escrows e ON e.job_id = j.id
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
			&application.BudgetLamports,
			&application.Funded,
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

// ListJobsByClient returns a client's jobs, newest first.
func ListJobsByClient(
	ctx context.Context,
	conn *sql.DB,
	clientID string,
) ([]Job, error) {
	const query = `
		SELECT
			j.id::text,
			j.solana_job_id::text,
			j.client_id::text,
			j.title,
			j.description,
			j.budget_lamports::text,
			COALESCE(e.funded_amount_lamports::text, ''),
			(e.job_id IS NOT NULL),
			j.created_at
		FROM jobs j
		LEFT JOIN job_escrows e ON e.job_id = j.id
		WHERE j.client_id = $1
		ORDER BY j.created_at DESC
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
			&job.SolanaJobID,
			&job.ClientID,
			&job.Title,
			&job.Description,
			&job.BudgetLamports,
			&job.FundedAmountLamports,
			&job.Funded,
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

// RecordEscrowFunding should be called only after the transaction has been
// verified on chain and its amount has been checked against the job budget.
func RecordEscrowFunding(
	ctx context.Context,
	conn *sql.DB,
	jobID string,
	fundedAmountLamports int64,
) error {
	if fundedAmountLamports <= 0 {
		return ErrFundedAmountPositive
	}

	result, err := conn.ExecContext(
		ctx,
		`UPDATE job_escrows
		 SET funded_amount_lamports = $2
		 WHERE job_id = $1
		   AND funded_amount_lamports IS NULL`,
		jobID,
		fundedAmountLamports,
	)
	if err != nil {
		return err
	}

	rowsUpdated, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsUpdated != 1 {
		return fmt.Errorf("job escrow row not found or already funded")
	}

	return nil
}