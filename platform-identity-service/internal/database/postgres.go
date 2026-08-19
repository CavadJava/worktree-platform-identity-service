package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
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
	var usersTableExists bool
	if err := db.QueryRow(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_name = 'users' AND table_schema = current_schema()
		)
	`).Scan(&usersTableExists); err != nil {
		return fmt.Errorf("check existing users table: %w", err)
	}

	if usersTableExists {
		var userCount int
		if err := db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&userCount); err != nil {
			return fmt.Errorf("count existing users: %w", err)
		}
		if userCount > 0 && os.Getenv("ALLOW_DESTRUCTIVE_MIGRATE") != "true" {
			return fmt.Errorf("refusing to run destructive migration against a database with existing user data — set ALLOW_DESTRUCTIVE_MIGRATE=true to override")
		}
	}

	_, err := db.Exec(`
		DROP TABLE IF EXISTS user_product_subscriptions;
		DROP TABLE IF EXISTS products;
		DROP TABLE IF EXISTS user_shop_memberships;
		DROP TABLE IF EXISTS users;
		DROP TABLE IF EXISTS shops;
		DROP TABLE IF EXISTS shop_roles;
		DROP TABLE IF EXISTS system_roles;
		DROP TABLE IF EXISTS roles;
		DROP TABLE IF EXISTS projects;

		CREATE TABLE system_roles (
			id SMALLSERIAL PRIMARY KEY,
			name TEXT UNIQUE NOT NULL
		);
		INSERT INTO system_roles (id, name) VALUES (1, 'superadmin'), (2, 'admin'), (3, 'user')
		ON CONFLICT (id) DO NOTHING;

		CREATE TABLE shop_roles (
			id SMALLSERIAL PRIMARY KEY,
			name TEXT UNIQUE NOT NULL
		);
		INSERT INTO shop_roles (id, name) VALUES (1, 'shop-admin'), (2, 'shop-user')
		ON CONFLICT (id) DO NOTHING;

		CREATE TABLE shops (
			id UUID PRIMARY KEY,
			name TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);

		CREATE TABLE users (
			id UUID PRIMARY KEY,
			name TEXT NOT NULL,
			username TEXT NOT NULL UNIQUE,
			email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			system_role_id SMALLINT NOT NULL REFERENCES system_roles(id) DEFAULT 3,
			status TEXT NOT NULL DEFAULT 'ACTIVE',
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);

		CREATE TABLE user_shop_memberships (
			id UUID PRIMARY KEY,
			user_id UUID NOT NULL REFERENCES users(id),
			shop_id UUID NOT NULL REFERENCES shops(id),
			shop_role_id SMALLINT NOT NULL REFERENCES shop_roles(id),
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			UNIQUE (user_id, shop_id)
		);
		CREATE INDEX idx_memberships_shop_id ON user_shop_memberships (shop_id);
		CREATE INDEX idx_memberships_user_id ON user_shop_memberships (user_id);

		CREATE TABLE products (
			id UUID PRIMARY KEY,
			name TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);

		CREATE TABLE user_product_subscriptions (
			id UUID PRIMARY KEY,
			user_id UUID NOT NULL REFERENCES users(id),
			product_id UUID NOT NULL REFERENCES products(id),
			subscripted BOOLEAN NOT NULL DEFAULT false,
			renewed BOOLEAN NOT NULL DEFAULT false,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			UNIQUE (user_id, product_id)
		);
	`)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}

// SeedSuperadmin ensures exactly one bootstrap superadmin account exists.
// Idempotent: safe to call on every startup. passwordHash must already be
// bcrypt-hashed by the caller.
func SeedSuperadmin(db *sql.DB, id, username, passwordHash string) error {
	_, err := db.Exec(`
		INSERT INTO users (id, name, username, email, password_hash, system_role_id, status, created_at, updated_at)
		VALUES ($1, 'Superadmin', $2, $2 || '@platform-identity.local', $3, 1, 'ACTIVE', now(), now())
		ON CONFLICT (username) DO NOTHING
	`, id, username, passwordHash)
	if err != nil {
		return fmt.Errorf("seed superadmin: %w", err)
	}
	return nil
}
