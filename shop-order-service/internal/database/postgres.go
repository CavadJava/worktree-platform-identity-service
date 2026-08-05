package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"shop-order-service/internal/config"
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

// Migrate owns the `orders` and `order_items` tables. It also relies on
// `users.user_seq` (registration-service), `shops.shop_seq` (shop-service),
// and `products` (shop-product-service) being present already — this
// service only reads those, via the shared Postgres instance, never writes
// to them.
func Migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS orders (
			id UUID PRIMARY KEY,
			order_number VARCHAR(50) NOT NULL UNIQUE,
			user_id UUID NOT NULL,
			shop_id UUID NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'pending',
			total_amount NUMERIC(12,2) NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders (user_id);
		CREATE INDEX IF NOT EXISTS idx_orders_shop_id ON orders (shop_id);

		CREATE TABLE IF NOT EXISTS order_items (
			id UUID PRIMARY KEY,
			order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
			product_id UUID NOT NULL,
			product_name VARCHAR(255) NOT NULL,
			unit_price NUMERIC(12,2) NOT NULL,
			quantity INTEGER NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON order_items (order_id);
	`)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}
