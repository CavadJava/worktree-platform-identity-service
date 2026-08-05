package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"shop-product-service/internal/config"
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
		CREATE TABLE IF NOT EXISTS products (
			id UUID PRIMARY KEY,
			shop_id UUID NOT NULL,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			price NUMERIC(12,2) NOT NULL DEFAULT 0,
			stock INTEGER NOT NULL DEFAULT 0,
			product_type_id UUID,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_products_shop_id ON products (shop_id);
		ALTER TABLE products ADD COLUMN IF NOT EXISTS product_type_id UUID;
		ALTER TABLE products ADD COLUMN IF NOT EXISTS brand VARCHAR(255);
		ALTER TABLE products ADD COLUMN IF NOT EXISTS material VARCHAR(255);
		ALTER TABLE products ADD COLUMN IF NOT EXISTS weight_kg NUMERIC(10,3);
		ALTER TABLE products ADD COLUMN IF NOT EXISTS origin_country VARCHAR(100);
		ALTER TABLE products ADD COLUMN IF NOT EXISTS warranty_months INTEGER;

		CREATE TABLE IF NOT EXISTS product_types (
			id UUID PRIMARY KEY,
			shop_id UUID NOT NULL,
			name VARCHAR(255) NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_product_types_shop_id ON product_types (shop_id);

		CREATE TABLE IF NOT EXISTS product_subtypes (
			id UUID PRIMARY KEY,
			product_type_id UUID NOT NULL REFERENCES product_types(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_product_subtypes_type_id ON product_subtypes (product_type_id);

		CREATE TABLE IF NOT EXISTS product_favorites (
			user_id UUID NOT NULL,
			product_id UUID NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			PRIMARY KEY (user_id, product_id)
		);
		CREATE INDEX IF NOT EXISTS idx_product_favorites_user_id ON product_favorites (user_id);

		CREATE TABLE IF NOT EXISTS product_items (
			id UUID PRIMARY KEY,
			product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			price NUMERIC(12,2) NOT NULL DEFAULT 0,
			stock INTEGER NOT NULL DEFAULT 0,
			is_discounted BOOLEAN NOT NULL DEFAULT false,
			discount_price NUMERIC(12,2),
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_product_items_product_id ON product_items (product_id);
	`)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}
