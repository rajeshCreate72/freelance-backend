package db

import (
	"context"
	"database/sql"
)

// LoginAccount contains the data needed to verify a login and create a token.
type LoginAccount struct {
	ID string
	Role string
	PasswordHash string
}

// FindLoginAccounts finds a client or freelancer by their unique email address.
func FindLoginAccounts (ctx context.Context, conn *sql.DB, email string) (LoginAccount, error) {
	// Find the account in either role-specific table through the shared emails table.
	query := `
		SELECT c.id::text, e.role::text, c.password_hash
		FROM emails e
		JOIN clients c ON c.email_id = e.id
		WHERE e.email = $1

		UNION ALL

		SELECT f.id::text, e.role::text, f.password_hash
		FROM emails e
		JOIN freelancers f ON f.email_id = e.id
		where e.email = $1
	`

	var account LoginAccount

	// Scan the selected database columns into the account struct.
	err := conn.QueryRowContext(ctx, query, email).Scan(
		&account.ID,
		&account.Role,
		&account.PasswordHash,
	)

	if err != nil {
		return LoginAccount{}, err
	}

	return account, nil
}
