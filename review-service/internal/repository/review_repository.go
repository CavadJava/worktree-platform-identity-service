package repository

import (
	"context"
	"database/sql"
	"errors"

	"review-service/internal/models"
)

var ErrReviewNotFound = errors.New("review not found")

type ReviewRepository struct {
	db *sql.DB
}

func NewReviewRepository(db *sql.DB) *ReviewRepository {
	return &ReviewRepository{db: db}
}

func (r *ReviewRepository) Create(ctx context.Context, rv *models.Review) error {
	const q = `
		INSERT INTO reviews (id, product_id, user_id, rating, text, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.ExecContext(ctx, q, rv.ID, rv.ProductID, rv.UserID, rv.Rating, rv.Text, rv.CreatedAt, rv.UpdatedAt)
	return err
}

func (r *ReviewRepository) FindByID(ctx context.Context, id string) (*models.Review, error) {
	const q = `SELECT id, product_id, user_id, rating, text, created_at, updated_at FROM reviews WHERE id = $1`
	rv := &models.Review{}
	err := r.db.QueryRowContext(ctx, q, id).Scan(&rv.ID, &rv.ProductID, &rv.UserID, &rv.Rating, &rv.Text, &rv.CreatedAt, &rv.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrReviewNotFound
	}
	if err != nil {
		return nil, err
	}
	return rv, nil
}

// ListByProduct returns a product's reviews newest-first, each with its
// media attached — two queries (reviews, then their media in one IN query)
// rather than a join, since a review can carry several media rows.
func (r *ReviewRepository) ListByProduct(ctx context.Context, productID string, limit int) ([]*models.Review, error) {
	const q = `
		SELECT id, product_id, user_id, rating, text, created_at, updated_at
		FROM reviews WHERE product_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`
	rows, err := r.db.QueryContext(ctx, q, productID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reviews := []*models.Review{}
	ids := make([]string, 0)
	byID := map[string]*models.Review{}
	for rows.Next() {
		rv := &models.Review{Media: []models.ReviewMedia{}}
		if err := rows.Scan(&rv.ID, &rv.ProductID, &rv.UserID, &rv.Rating, &rv.Text, &rv.CreatedAt, &rv.UpdatedAt); err != nil {
			return nil, err
		}
		reviews = append(reviews, rv)
		ids = append(ids, rv.ID)
		byID[rv.ID] = rv
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return reviews, nil
	}

	mediaRows, err := r.db.QueryContext(ctx, `
		SELECT id, review_id, media_type, url, created_at
		FROM review_media WHERE review_id::text = ANY($1)
		ORDER BY created_at ASC
	`, ids)
	if err != nil {
		return nil, err
	}
	defer mediaRows.Close()

	for mediaRows.Next() {
		m := models.ReviewMedia{}
		if err := mediaRows.Scan(&m.ID, &m.ReviewID, &m.MediaType, &m.URL, &m.CreatedAt); err != nil {
			return nil, err
		}
		if rv, ok := byID[m.ReviewID]; ok {
			rv.Media = append(rv.Media, m)
		}
	}
	return reviews, mediaRows.Err()
}

func (r *ReviewRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM reviews WHERE id = $1`, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrReviewNotFound
	}
	return nil
}

func (r *ReviewRepository) AddMedia(ctx context.Context, m *models.ReviewMedia) error {
	const q = `INSERT INTO review_media (id, review_id, media_type, url, created_at) VALUES ($1, $2, $3, $4, $5)`
	_, err := r.db.ExecContext(ctx, q, m.ID, m.ReviewID, m.MediaType, m.URL, m.CreatedAt)
	return err
}
