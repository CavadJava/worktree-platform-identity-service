package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"localization-service/internal/models"
)

var ErrTranslationNotFound = errors.New("translation not found")

type TranslationRepository struct {
	db *sql.DB
}

func NewTranslationRepository(db *sql.DB) *TranslationRepository {
	return &TranslationRepository{db: db}
}

// Upsert inserts a new translation, or — if one already exists for this
// (namespace, key, locale) — updates its value in place, keeping the
// original id/created_at. t is updated in place to reflect what's stored.
func (r *TranslationRepository) Upsert(ctx context.Context, t *models.Translation) error {
	const q = `
		INSERT INTO translations (id, namespace, key, locale, value, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (namespace, key, locale)
		DO UPDATE SET value = EXCLUDED.value, updated_at = EXCLUDED.updated_at
		RETURNING id, created_at
	`
	return r.db.QueryRowContext(ctx, q, t.ID, t.Namespace, t.Key, t.Locale, t.Value, t.CreatedAt, t.UpdatedAt).
		Scan(&t.ID, &t.CreatedAt)
}

func (r *TranslationRepository) Get(ctx context.Context, namespace, key, locale string) (*models.Translation, error) {
	const q = `
		SELECT id, namespace, key, locale, value, created_at, updated_at
		FROM translations WHERE namespace = $1 AND key = $2 AND locale = $3
	`
	t := &models.Translation{}
	err := r.db.QueryRowContext(ctx, q, namespace, key, locale).Scan(
		&t.ID, &t.Namespace, &t.Key, &t.Locale, &t.Value, &t.CreatedAt, &t.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrTranslationNotFound
	}
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (r *TranslationRepository) List(ctx context.Context, namespace, locale, key string) ([]*models.Translation, error) {
	q := `SELECT id, namespace, key, locale, value, created_at, updated_at FROM translations WHERE 1=1`
	args := []interface{}{}

	if namespace != "" {
		args = append(args, namespace)
		q += fmt.Sprintf(" AND namespace = $%d", len(args))
	}
	if locale != "" {
		args = append(args, locale)
		q += fmt.Sprintf(" AND locale = $%d", len(args))
	}
	if key != "" {
		args = append(args, key)
		q += fmt.Sprintf(" AND key = $%d", len(args))
	}
	q += ` ORDER BY namespace, key, locale`

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	translations := []*models.Translation{}
	for rows.Next() {
		t := &models.Translation{}
		if err := rows.Scan(&t.ID, &t.Namespace, &t.Key, &t.Locale, &t.Value, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		translations = append(translations, t)
	}
	return translations, rows.Err()
}

func (r *TranslationRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM translations WHERE id = $1`, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrTranslationNotFound
	}
	return nil
}

func (r *TranslationRepository) ListLocales(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT DISTINCT locale FROM translations ORDER BY locale`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	locales := []string{}
	for rows.Next() {
		var locale string
		if err := rows.Scan(&locale); err != nil {
			return nil, err
		}
		locales = append(locales, locale)
	}
	return locales, rows.Err()
}
