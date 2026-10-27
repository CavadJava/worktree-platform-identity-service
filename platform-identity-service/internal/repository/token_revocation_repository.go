package repository

import (
	"context"
	"database/sql"
	"time"
)

type TokenRevocationRepository struct {
	db *sql.DB
}

func NewTokenRevocationRepository(db *sql.DB) *TokenRevocationRepository {
	return &TokenRevocationRepository{db: db}
}

// Revoke marks jti as logged out. expiresAt should be the token's own exp
// claim — stored for a possible future cleanup job, not required for
// correctness today. ON CONFLICT DO NOTHING makes a repeated call for the
// same jti (e.g. a retried logout request) a harmless no-op.
func (r *TokenRevocationRepository) Revoke(ctx context.Context, jti string, expiresAt time.Time) error {
	const q = `
		INSERT INTO token_revocations (jti, expires_at) VALUES ($1, $2)
		ON CONFLICT (jti) DO NOTHING
	`
	_, err := r.db.ExecContext(ctx, q, jti, expiresAt)
	return err
}

// IsRevoked reports whether jti has been logged out.
func (r *TokenRevocationRepository) IsRevoked(ctx context.Context, jti string) (bool, error) {
	const q = `SELECT 1 FROM token_revocations WHERE jti = $1`
	var one int
	err := r.db.QueryRowContext(ctx, q, jti).Scan(&one)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
