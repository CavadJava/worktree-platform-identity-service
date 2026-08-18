package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"platform-identity-service/internal/config"
)

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
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS projects (
			id UUID PRIMARY KEY,
			name TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);

		CREATE TABLE IF NOT EXISTS roles (
			id SMALLSERIAL PRIMARY KEY,
			name TEXT UNIQUE NOT NULL
		);

		INSERT INTO roles (id, name) VALUES (1, 'user'), (2, 'admin'), (3, 'superadmin')
		ON CONFLICT (id) DO NOTHING;

		CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY,
			name TEXT NOT NULL,
			username TEXT NOT NULL UNIQUE,
			email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			project_id UUID REFERENCES projects(id),
			role_id SMALLINT REFERENCES roles(id),
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);

		CREATE INDEX IF NOT EXISTS idx_users_project_id ON users (project_id);
	`)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}

// SeedSuperadmin ensures exactly one bootstrap superadmin account exists,
// with no project (system-level). Idempotent: safe to call on every
// startup. passwordHash must already be bcrypt-hashed by the caller —
// hashing is not this package's concern, it only persists what it's given.
func SeedSuperadmin(db *sql.DB, id, username, passwordHash string) error {
	_, err := db.Exec(`
		INSERT INTO users (id, name, username, email, password_hash, project_id, role_id, created_at, updated_at)
		VALUES ($1, 'Superadmin', $2, $2 || '@platform-identity.local', $3, NULL, 3, now(), now())
		ON CONFLICT (username) DO NOTHING
	`, id, username, passwordHash)
	if err != nil {
		return fmt.Errorf("seed superadmin: %w", err)
	}
	return nil
}
