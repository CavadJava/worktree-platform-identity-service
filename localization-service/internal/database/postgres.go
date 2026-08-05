package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"

	"localization-service/internal/config"
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

// Migrate owns the `translations` table and seeds it with translations for
// the response-envelope error codes already used across every other
// service (bad_request, unauthorized, forbidden, not_found, conflict,
// internal_error, service_unavailable, error) — the most immediately
// useful content for a first integration, since any client can already
// look these codes up from an error response and resolve a localized
// message here.
func Migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS translations (
			id UUID PRIMARY KEY,
			namespace VARCHAR(100) NOT NULL,
			key VARCHAR(150) NOT NULL,
			locale VARCHAR(10) NOT NULL,
			value TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			UNIQUE (namespace, key, locale)
		);
		CREATE INDEX IF NOT EXISTS idx_translations_lookup ON translations (namespace, locale);
	`)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	if err := seedErrorCodes(db); err != nil {
		return fmt.Errorf("seed: %w", err)
	}
	return nil
}

type seedEntry struct {
	key    string
	locale string
	value  string
}

func seedErrorCodes(db *sql.DB) error {
	entries := []seedEntry{
		{"bad_request", "az", "Sorğu düzgün deyil"},
		{"bad_request", "en", "Bad request"},
		{"bad_request", "ru", "Некорректный запрос"},

		{"unauthorized", "az", "Giriş tələb olunur"},
		{"unauthorized", "en", "Unauthorized"},
		{"unauthorized", "ru", "Требуется авторизация"},

		{"forbidden", "az", "İcazə yoxdur"},
		{"forbidden", "en", "Forbidden"},
		{"forbidden", "ru", "Доступ запрещён"},

		{"not_found", "az", "Tapılmadı"},
		{"not_found", "en", "Not found"},
		{"not_found", "ru", "Не найдено"},

		{"conflict", "az", "Artıq mövcuddur"},
		{"conflict", "en", "Conflict"},
		{"conflict", "ru", "Конфликт"},

		{"internal_error", "az", "Daxili xəta baş verdi"},
		{"internal_error", "en", "Internal error"},
		{"internal_error", "ru", "Внутренняя ошибка"},

		{"service_unavailable", "az", "Servis əlçatan deyil"},
		{"service_unavailable", "en", "Service unavailable"},
		{"service_unavailable", "ru", "Сервис недоступен"},

		{"error", "az", "Naməlum xəta"},
		{"error", "en", "Unknown error"},
		{"error", "ru", "Неизвестная ошибка"},
	}

	const q = `
		INSERT INTO translations (id, namespace, key, locale, value, created_at, updated_at)
		VALUES ($1, 'errors', $2, $3, $4, now(), now())
		ON CONFLICT (namespace, key, locale) DO NOTHING
	`
	for _, e := range entries {
		if _, err := db.Exec(q, uuid.NewString(), e.key, e.locale, e.value); err != nil {
			return err
		}
	}
	return nil
}
