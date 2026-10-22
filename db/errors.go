package db

import (
	"errors"
	"github.com/jackc/pgx/v5/pgconn"
)

var ErrEmailAlreadyRegistered = errors.New("Email already registered")
var (
	ErrJobNotFound = errors.New("Job not found")
	ErrApplicationAlreadyExists = errors.New("Freelancer already applied for the job")
)

var (
	ErrJobApplicationNotFound = errors.New("Job or application not found")
	ErrJobAlreadyAssigned     = errors.New("Job already has an assignment")
	ErrAssignmentNotPending   = errors.New("No pending assignment for this freelancer and job")
)

var (
	ErrReviewNotAllowed        = errors.New("account cannot review this job")
	ErrReviewAlreadySubmitted  = errors.New("review already submitted for this job")
)

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	
	if errors.As(err, &pgErr) {
		// Postgresql unique violation code
		return pgErr.Code == "23505"
	} 

	return false
}

