package db

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

func SaveRefreshToken(
	ctx context.Context,
	conn *sql.DB,
	tokenHash string,
	accountID string,
	role string,
	expiresAt time.Time,
) error {
	const query = `
		INSERT INTO refresh_tokens (token_hash, account_id, role, expires_at)
		VALUES ($1, $2, $3, $4)
	`

	_, err := conn.ExecContext(ctx, query, tokenHash, accountID, role, expiresAt)

	return err
}

func RotateRefreshToken(
	ctx context.Context,
	conn *sql.DB,
	oldHash string,
	newHash string,
	newExpiry time.Time,
) (string, string, error) {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return "", "", err
	}
	defer tx.Rollback()

	var accountID string
	var role string
	
	const findQuery = `
		SELECT account_id::text, role::text
		FROM refresh_tokens
		WHERE token_hash = $1
		  AND revoked_at IS NULL
		  AND expires_at > NOW()
		FOR UPDATE
	`
	// Finding the old token from the refresh_tokens table, oldHash is the old token
	// Assigning the accountID and role to the variables
	err = tx.QueryRowContext(ctx, findQuery, oldHash).Scan(&accountID, &role)

	if errors.Is(err, sql.ErrNoRows) {
		return "", "", ErrInvalidRefreshToken
	}
	if err != nil {
		return "", "", err
	}

	const revokeQuery = `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE token_hash = $1
	`

	if _, err := tx.ExecContext(ctx, revokeQuery, oldHash); err != nil {
		return "", "", err
	}

	const insertQuery = `
		INSERT INTO refresh_tokens (token_hash, account_id, role, expires_at)
		VALUES ($1, $2, $3, $4)
	`

	if _, err := tx.ExecContext(
		ctx, insertQuery, newHash, accountID, role, newExpiry,
	); err != nil {
		return "", "", err
	}

	if err := tx.Commit(); err != nil {
		return "", "", err
	}

	return accountID, role, nil
}
