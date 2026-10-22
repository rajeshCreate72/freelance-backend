package db

import (
	"context"
	"database/sql"
	"time"
)

type ClientProfile struct {
	ID string `json:"id"`
	Username string `json:"username"`
	Email string `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

func CreateClient(conn *sql.DB, username, email, passwordHash string) error {
	ctx := context.Background()

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	var emailID string

	emailQuery := `
		INSERT INTO emails (email, role)
		VALUES ($1, 'client')
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

	clientQuery := `
			INSERT INTO clients (username, email_id, password_hash)
			VALUES($1, $2, $3)
	`
	_, err = tx.ExecContext(
		context.Background(),
		clientQuery,
		username,
		emailID,
		passwordHash,
	)

	if err !=  nil {
		return err
	}

	return tx.Commit()
}

func GetClientByID(ctx context.Context, conn *sql.DB, clientID string) (ClientProfile, error) {
	const query = `
		SELECT c.id::text, c.username, e.email, c.created_at
		FROM clients c
		JOIN email e ON e.id = c.email_id
		WHERE c.id = $1
	`
	var profile ClientProfile

	err := conn.QueryRowContext(ctx, query, clientID).Scan(
		&profile.ID,
		&profile.Username,
		&profile.Email,
		&profile.CreatedAt,
	)

	if err != nil {
		return ClientProfile{}, err
	}

	return profile, nil
}