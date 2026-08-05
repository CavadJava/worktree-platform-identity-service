package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"log-service/internal/config"
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
	const schema = `
		CREATE TABLE IF NOT EXISTS service_logs (
			id UUID PRIMARY KEY,
			service VARCHAR(100) NOT NULL,
			level VARCHAR(20) NOT NULL DEFAULT 'info',
			method VARCHAR(10),
			path VARCHAR(500),
			status INT,
			duration_ms BIGINT,
			message TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_service_logs_service_created ON service_logs (service, created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_service_logs_level ON service_logs (level);
		CREATE INDEX IF NOT EXISTS idx_service_logs_created ON service_logs (created_at DESC);
	`
	_, err := db.Exec(schema)
	return err
}
