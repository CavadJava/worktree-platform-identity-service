package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"shop-service/internal/config"
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

// Migrate owns the `shops` and `shop_applications` tables. It also relies on
// `users.shop_id` / `users.shop_role_level` (owned by registration-service)
// being present already — approval writes to those two columns only.
func Migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS shops (
			id UUID PRIMARY KEY,
			owner_id UUID NOT NULL,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			temporary BOOLEAN NOT NULL DEFAULT false,
			shop_seq BIGSERIAL UNIQUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_shops_owner_id ON shops (owner_id);
		ALTER TABLE shops ADD COLUMN IF NOT EXISTS temporary BOOLEAN NOT NULL DEFAULT false;
		ALTER TABLE shops ADD COLUMN IF NOT EXISTS shop_seq BIGSERIAL UNIQUE;

		CREATE TABLE IF NOT EXISTS shop_applications (
			id UUID PRIMARY KEY,
			applicant_id UUID NOT NULL,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			template_version VARCHAR(20) NOT NULL DEFAULT 'v1',
			status VARCHAR(20) NOT NULL DEFAULT 'pending',
			reviewer_id UUID,
			review_note TEXT,
			shop_id UUID,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_shop_applications_status ON shop_applications (status);
		CREATE INDEX IF NOT EXISTS idx_shop_applications_applicant ON shop_applications (applicant_id);

		CREATE TABLE IF NOT EXISTS shop_subscriptions (
			user_id UUID NOT NULL,
			shop_id UUID NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			PRIMARY KEY (user_id, shop_id)
		);
		CREATE INDEX IF NOT EXISTS idx_shop_subscriptions_shop_id ON shop_subscriptions (shop_id);
	`)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}
