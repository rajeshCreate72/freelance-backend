package db

import (
	"context"
	"database/sql"
	"time"
)

// CreateWalletNonce saves the nonce associated with an authenticated account
// and the wallet address it intends to connect.
func CreateWalletNonce(
	ctx context.Context,
	conn *sql.DB,
	accountID string,
	role string,
	walletAddress string,
	nonce string,
	expiresAt time.Time,
) error {
	const query = `
		INSERT INTO wallet_nonces
			(account_id, role, wallet_address, nonce, expires_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := conn.ExecContext(
		ctx,
		query,
		accountID,
		role,
		walletAddress,
		nonce,
		expiresAt,
	)
	return err
}

// ConsumeWalletNonce succeeds only once, and only before the nonce expires.
// Call this from the wallet-signature verification flow.
func ConsumeWalletNonce(
	ctx context.Context,
	conn *sql.DB,
	accountID string,
	role string,
	walletAddress string,
	nonce string,
) error {
	const query = `
		UPDATE wallet_nonces
		SET consumed_at = NOW()
		WHERE account_id = $1
		AND role = $2
		AND wallet_address = $3
		AND nonce = $4
		AND consumed_at IS NULL
		AND expires_at > NOW()
	`

	result, err := conn.ExecContext(
		ctx,
		query,
		accountID,
		role,
		walletAddress,
		nonce,
	)

	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrInvalidWalletNonce
	}

	return nil
}