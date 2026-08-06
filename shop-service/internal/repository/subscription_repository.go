package repository

import (
	"context"
	"database/sql"
	"time"

	"shop-service/internal/models"
)

type SubscriptionRepository struct {
	db *sql.DB
}

func NewSubscriptionRepository(db *sql.DB) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

func (r *SubscriptionRepository) Add(ctx context.Context, userID, shopID string) error {
	const q = `
		INSERT INTO shop_subscriptions (user_id, shop_id, created_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, shop_id) DO NOTHING
	`
	_, err := r.db.ExecContext(ctx, q, userID, shopID, time.Now().UTC())
	return err
}

func (r *SubscriptionRepository) Remove(ctx context.Context, userID, shopID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM shop_subscriptions WHERE user_id = $1 AND shop_id = $2`, userID, shopID)
	return err
}

// ListByShop returns a shop's subscribers (just user_id + when), newest first.
func (r *SubscriptionRepository) ListByShop(ctx context.Context, shopID string) ([]*models.Subscriber, error) {
	const q = `
		SELECT user_id, created_at FROM shop_subscriptions
		WHERE shop_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, q, shopID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	subscribers := []*models.Subscriber{}
	for rows.Next() {
		sub := &models.Subscriber{}
		if err := rows.Scan(&sub.UserID, &sub.CreatedAt); err != nil {
			return nil, err
		}
		subscribers = append(subscribers, sub)
	}
	return subscribers, rows.Err()
}

// ListByUser returns the full shop rows a user is subscribed to, newest first.
func (r *SubscriptionRepository) ListByUser(ctx context.Context, userID string) ([]*models.Shop, error) {
	const q = `
		SELECT s.id, s.owner_id, s.name, COALESCE(s.description, ''), s.temporary, s.created_at, s.updated_at
		FROM shop_subscriptions sub
		JOIN shops s ON s.id = sub.shop_id
		WHERE sub.user_id = $1
		ORDER BY sub.created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	shops := []*models.Shop{}
	for rows.Next() {
		s := &models.Shop{}
		if err := rows.Scan(&s.ID, &s.OwnerID, &s.Name, &s.Description, &s.Temporary, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		shops = append(shops, s)
	}
	return shops, rows.Err()
}
