package db

import (
	"context"
	"database/sql"
	"time"
)


type FreelancerProfile struct {
	ID string `json:"id"`
	Username string `json:"username"`
	Email string `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

func CreateFreelancer(
	conn *sql.DB,
	username string,
	email string,
	passwordHash string,
) error {
	ctx := context.Background()

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	var emailID string

	emailQuery := `
		INSERT INTO emails (email, role)
		VALUES ($1, 'freelancer')
		RETURNING id
	`

	err = tx.QueryRowContext(
		ctx,
		emailQuery,
		email,
	).Scan(&emailID)

	if err != nil {
		if isUniqueViolation(err) {
			return ErrEmailAlreadyRegistered
		}
		return err
	}

	freelancerQuery := `
		INSERT INTO freelancers (username, email_id, password_hash)
		VALUES ($1, $2, $3)
	`

	_, err = tx.ExecContext(
		ctx,
		freelancerQuery,
		username,
		emailID,
		passwordHash,
	)

	if err != nil {
		return err
	}

	return tx.Commit()
}

// GetFreelanceByID finds a freelancer by the ID stored in the JWT.
func GetFreelanceByID(
	ctx context.Context,
	conn *sql.DB,
	freelacerID string,
) (FreelancerProfile, error) {
	const query = `
		SELECT f.id::text, f.username, e.email, f.created_at
		FROM freelancers f
		JOIN emails e ON e.id = f.email_id
		WHERE f.id = $1
	`

	var profile FreelancerProfile

	err := conn.QueryRowContext(ctx, query, freelacerID).Scan(
		&profile.ID,
		&profile.Username,
		&profile.Email,
		&profile.CreatedAt,
	)
	if err != nil {
		return FreelancerProfile{}, err
	}

	return profile, nil
}
