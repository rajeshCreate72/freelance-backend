package db

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

var ErrEmailAlreadyRegistered = errors.New("Email already registered")

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	
	if errors.As(err, &pgErr) {
		// Postgresql unique violation code
		return pgErr.Code == "23505"
	} 

	return false
}