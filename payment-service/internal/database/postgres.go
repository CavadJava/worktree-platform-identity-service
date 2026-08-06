package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"payment-service/internal/config"
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

// Migrate owns `payments`, `payment_items` and `shop_balances`. Relies on
// `orders` (shop-order-service) only loosely — order_id is stored but never
// joined/read back from here, matching the "snapshot everything needed at
// write time" convention shop-order-service itself uses for products.
func Migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS payments (
			id UUID PRIMARY KEY,
			order_id UUID NOT NULL,
			user_id UUID NOT NULL,
			shop_id UUID NOT NULL,
			amount NUMERIC(12,2) NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'completed',
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		-- one payment per order: shop-order-service calls POST /payments exactly
		-- once, right after creating the order.
		CREATE UNIQUE INDEX IF NOT EXISTS idx_payments_order_id ON payments (order_id);
		CREATE INDEX IF NOT EXISTS idx_payments_shop_created ON payments (shop_id, created_at DESC);

		CREATE TABLE IF NOT EXISTS payment_items (
			id UUID PRIMARY KEY,
			payment_id UUID NOT NULL REFERENCES payments(id) ON DELETE CASCADE,
			product_id UUID NOT NULL,
			product_name VARCHAR(255) NOT NULL,
			quantity INTEGER NOT NULL,
			unit_price NUMERIC(12,2) NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_payment_items_payment_id ON payment_items (payment_id);

		CREATE TABLE IF NOT EXISTS shop_balances (
			shop_id UUID PRIMARY KEY,
			balance NUMERIC(12,2) NOT NULL DEFAULT 0,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
	`)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}
