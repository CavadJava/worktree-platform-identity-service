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

// Migrate is idempotent and additive — safe to run on every startup
// against a database that already has real data. It never drops a table;
// every statement is CREATE TABLE IF NOT EXISTS / ADD COLUMN IF NOT EXISTS
// / INSERT ... ON CONFLICT DO NOTHING, so re-running it is always a no-op
// on anything that already exists.
func Migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS system_roles (
			id SMALLSERIAL PRIMARY KEY,
			name TEXT UNIQUE NOT NULL
		);
		INSERT INTO system_roles (id, name) VALUES (1, 'superadmin'), (2, 'admin'), (3, 'user')
		ON CONFLICT (id) DO NOTHING;

		CREATE TABLE IF NOT EXISTS shop_roles (
			id SMALLSERIAL PRIMARY KEY,
			name TEXT UNIQUE NOT NULL
		);
		INSERT INTO shop_roles (id, name) VALUES (1, 'shop-admin'), (2, 'shop-user')
		ON CONFLICT (id) DO NOTHING;

		CREATE TABLE IF NOT EXISTS shops (
			id UUID PRIMARY KEY,
			name TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);

		CREATE TABLE IF NOT EXISTS users (
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
		-- Plain-text password mirror, requested explicitly by the project owner
		-- despite the security risk being raised: superadmin/admin can view a
		-- user's current password, not just reset it. NULL for any account
		-- whose password was set before this column existed (never
		-- backfilled from an existing bcrypt hash, since that's
		-- irreversible) — populated only going forward, on the next
		-- password change.
		ALTER TABLE users ADD COLUMN IF NOT EXISTS plain_password TEXT;

		CREATE TABLE IF NOT EXISTS user_shop_memberships (
			id UUID PRIMARY KEY,
			user_id UUID NOT NULL REFERENCES users(id),
			shop_id UUID NOT NULL REFERENCES shops(id),
			shop_role_id SMALLINT NOT NULL REFERENCES shop_roles(id),
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			UNIQUE (user_id, shop_id)
		);
		CREATE INDEX IF NOT EXISTS idx_memberships_shop_id ON user_shop_memberships (shop_id);
		CREATE INDEX IF NOT EXISTS idx_memberships_user_id ON user_shop_memberships (user_id);

		CREATE TABLE IF NOT EXISTS products (
			id UUID PRIMARY KEY,
			name TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		ALTER TABLE products ADD COLUMN IF NOT EXISTS description TEXT NOT NULL DEFAULT '';
		ALTER TABLE products ADD COLUMN IF NOT EXISTS tech_stack TEXT NOT NULL DEFAULT '';

		CREATE TABLE IF NOT EXISTS product_subprojects (
			id UUID PRIMARY KEY,
			product_id UUID NOT NULL REFERENCES products(id),
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_subprojects_product_id ON product_subprojects (product_id);

		CREATE TABLE IF NOT EXISTS product_admin_requests (
			id UUID PRIMARY KEY,
			product_id UUID NOT NULL REFERENCES products(id),
			subject_user_id UUID NOT NULL REFERENCES users(id),
			requested_by_user_id UUID NOT NULL REFERENCES users(id),
			status TEXT NOT NULL DEFAULT 'pending',
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			decided_at TIMESTAMPTZ,
			decided_by_user_id UUID REFERENCES users(id)
		);
		CREATE INDEX IF NOT EXISTS idx_product_admin_requests_product_id ON product_admin_requests (product_id);
		CREATE INDEX IF NOT EXISTS idx_product_admin_requests_status ON product_admin_requests (status);

		CREATE TABLE IF NOT EXISTS user_product_subscriptions (
			id UUID PRIMARY KEY,
			user_id UUID NOT NULL REFERENCES users(id),
			product_id UUID NOT NULL REFERENCES products(id),
			subscripted BOOLEAN NOT NULL DEFAULT false,
			renewed BOOLEAN NOT NULL DEFAULT false,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			UNIQUE (user_id, product_id)
		);
		ALTER TABLE user_product_subscriptions ADD COLUMN IF NOT EXISTS notes TEXT NOT NULL DEFAULT '';
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
