package repository

import (
	"context"
	"database/sql"
	"errors"

	"platform-identity-service/internal/models"
)

var ErrSubscriptionNotFound = errors.New("subscription not found")

type SubscriptionRepository struct {
	db *sql.DB
}

func NewSubscriptionRepository(db *sql.DB) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

// Upsert creates or updates a user's subscription row for a product.
func (r *SubscriptionRepository) Upsert(ctx context.Context, s *models.Subscription) error {
	const q = `
		INSERT INTO user_product_subscriptions (id, user_id, product_id, subscripted, renewed, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id, product_id) DO UPDATE
		SET subscripted = EXCLUDED.subscripted, renewed = EXCLUDED.renewed, updated_at = EXCLUDED.updated_at
	`
	_, err := r.db.ExecContext(ctx, q, s.ID, s.UserID, s.ProductID, s.Subscripted, s.Renewed, s.CreatedAt, s.UpdatedAt)
	return err
}

func (r *SubscriptionRepository) GetByUserAndProduct(ctx context.Context, userID, productID string) (*models.Subscription, error) {
	const q = `
		SELECT id, user_id, product_id, subscripted, renewed, created_at, updated_at
		FROM user_product_subscriptions WHERE user_id = $1 AND product_id = $2
	`
	var s models.Subscription
	err := r.db.QueryRowContext(ctx, q, userID, productID).Scan(
		&s.ID, &s.UserID, &s.ProductID, &s.Subscripted, &s.Renewed, &s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrSubscriptionNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}
