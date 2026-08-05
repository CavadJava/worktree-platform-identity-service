package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"user-service/internal/config"
)

// Connect opens a pool against the same Postgres instance/table that
// registration-service owns and migrates. For the `users` table this
// service only reads/writes existing columns; the `user_addresses` table
// below is owned (created/migrated) by this service.
func Connect(cfg *config.Config) (*sql.DB, error) {
	db, err := sql.Open("pgx", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	return db, nil
}

func Migrate(db *sql.DB) error {
	const schema = `
		CREATE TABLE IF NOT EXISTS user_addresses (
			id UUID PRIMARY KEY,
			user_id UUID NOT NULL,
			title VARCHAR(100) NOT NULL,
			full_address TEXT NOT NULL,
			city VARCHAR(100),
			phone VARCHAR(50),
			is_default BOOLEAN NOT NULL DEFAULT false,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_user_addresses_user_id ON user_addresses (user_id);
	`
	_, err := db.Exec(schema)
	return err
}
