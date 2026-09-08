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
		INSERT INTO user_product_subscriptions (id, user_id, product_id, subscripted, renewed, notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (user_id, product_id) DO UPDATE
		SET subscripted = EXCLUDED.subscripted, renewed = EXCLUDED.renewed, notes = EXCLUDED.notes, updated_at = EXCLUDED.updated_at
	`
	_, err := r.db.ExecContext(ctx, q, s.ID, s.UserID, s.ProductID, s.Subscripted, s.Renewed, s.Notes, s.CreatedAt, s.UpdatedAt)
	return err
}

// ListProductIDsByUser returns every product_id userID holds a subscription
// row for, regardless of Subscripted's value — an admin managing a
// product they were subscribed-then-unsubscribed from still manages it in
// this scoping sense (a separate concern from whether their own account
// has "full" product access, which CheckAccess covers).
func (r *SubscriptionRepository) ListProductIDsByUser(ctx context.Context, userID string) ([]string, error) {
	const q = `SELECT product_id FROM user_product_subscriptions WHERE user_id = $1`
	rows, err := r.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// ListByProduct returns every subscription row for productID, joined with
// each subscriber's identity fields — powers a product's "customers" list
// (any user with a subscription row, regardless of Subscripted's value, and
// regardless of their system_role — a product-scoped admin is just another
// subscriber from this table's point of view).
func (r *SubscriptionRepository) ListByProduct(ctx context.Context, productID string) ([]models.SubscriptionWithUser, error) {
	const q = `
		SELECT s.id, s.user_id, s.product_id, s.subscripted, s.renewed, s.notes, s.created_at, s.updated_at,
			u.name, u.username, u.email, sr.name
		FROM user_product_subscriptions s
		JOIN users u ON u.id = s.user_id
		JOIN system_roles sr ON sr.id = u.system_role_id
		WHERE s.product_id = $1
		ORDER BY s.created_at
	`
	rows, err := r.db.QueryContext(ctx, q, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	subs := []models.SubscriptionWithUser{}
	for rows.Next() {
		var s models.SubscriptionWithUser
		if err := rows.Scan(
			&s.ID, &s.UserID, &s.ProductID, &s.Subscripted, &s.Renewed, &s.Notes, &s.CreatedAt, &s.UpdatedAt,
			&s.UserName, &s.UserUsername, &s.UserEmail, &s.UserSystemRole,
		); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, rows.Err()
}

func (r *SubscriptionRepository) GetByUserAndProduct(ctx context.Context, userID, productID string) (*models.Subscription, error) {
	const q = `
		SELECT id, user_id, product_id, subscripted, renewed, notes, created_at, updated_at
		FROM user_product_subscriptions WHERE user_id = $1 AND product_id = $2
	`
	var s models.Subscription
	err := r.db.QueryRowContext(ctx, q, userID, productID).Scan(
		&s.ID, &s.UserID, &s.ProductID, &s.Subscripted, &s.Renewed, &s.Notes, &s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrSubscriptionNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}
