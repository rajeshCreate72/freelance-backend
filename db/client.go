package db

import (
	"context"
	"database/sql"
)

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
