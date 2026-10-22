package db

import (
	"context"
	"database/sql"
	"time"
)

type Assignment struct {
	JobID string `json:"job_id"`
	FreelancerID string `json:"freelancer_id"`
	Status string `json:"status"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AssignFreelancer selects an applicant for a job.
// It checks that the authenticated client owns the job and that the
// selected application belongs to that job
func AssignFreelancer(
	ctx context.Context,
	conn *sql.DB,
	jobID string,
	clientID string,
	applicationID string,
) (Assignment, error) {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return Assignment{}, err
	}
	defer tx.Rollback()

	const query = `
		SELECT a.freelancer_id::text
		FROM jobs as j
		JOIN applications AS a ON a.job_id = j.id
		WHERE j.id = $1
			AND j.client_id = $2
			AND a.id = $3
		FOR UPDATE OF j
	`

	var freelancerID string
	err = tx.QueryRowContext(ctx, query, jobID, clientID, applicationID).Scan(&freelancerID)
	if err == sql.ErrNoRows {
		return Assignment{}, ErrJobApplicationNotFound
	}
	if err != nil {
		return Assignment{}, err
	}

	const query_assignment = `
		INSERT INTO job_assignments (job_id, freelancer_id, status_id)
		SELECT $1, $2, id
		FROM assignment_statuses
		WHERE name = 'Assigned'
		RETURNING
			job_id::text,
			freelancer_id::text,
			(SELECT name
			 FROM assignment_statuses
			 WHERE id = job_assignments.status_id),
			updated_at
	`

	var assignment Assignment
	err = tx.QueryRowContext(
		ctx,
		query_assignment,
		jobID,
		freelancerID,
	).Scan(
		&assignment.JobID,
		&assignment.FreelancerID,
		&assignment.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return Assignment{}, ErrJobAlreadyAssigned
		}
		return Assignment{}, err
	}

	if err := tx.Commit(); err != nil {
		return Assignment{}, err
	}

	return assignment, nil
}

// AcceptAssignment lets only the selected freelancer accept the assignment
// It changes the backend assignment state from "Assigned" to "Accepted"
func AcceptAssignment(
	ctx context.Context,
	conn *sql.DB,
	jobID string,
	freelancerID string,
) (Assignment, error) {
	const query = `
		UPDATE job_assignments
		SET status_id = (
				SELECT id FROM assignment_statuses WHERE name = 'Accepted'
			),
			updated_at = CURRENT_TIMESTAMP
		WHERE job_id = $1
		  AND freelancer_id = $2
		  AND status_id = (
				SELECT id FROM assignment_statuses WHERE name = 'Assigned'
			)
		RETURNING
			job_id::text,
			freelancer_id::text,
			(SELECT name
			 FROM assignment_statuses
			 WHERE id = job_assignments.status_id),
			updated_at
	`

	var assignment Assignment
	err := conn.QueryRowContext(
		ctx,
		query,
		jobID,
		freelancerID,
	).Scan(
		&assignment.JobID,
		&assignment.FreelancerID,
		&assignment.UpdatedAt,
	)
	if err != sql.ErrNoRows {
		return Assignment{}, ErrAssignmentNotPending
	}
	if err != nil {
		return Assignment{}, err
	}

	return assignment, nil
}