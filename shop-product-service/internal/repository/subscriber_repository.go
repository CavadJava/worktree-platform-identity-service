package repository

import (
	"context"
	"database/sql"
)

// SubscriberRepository reads shop-service's `shop_subscriptions` and
// `shops` tables (shared-DB convention: simple, stable columns owned by
// another service are read directly instead of over HTTP) — used to fan
// out "new product" notifications to a shop's subscribers.
type SubscriberRepository struct {
	db *sql.DB
}

func NewSubscriberRepository(db *sql.DB) *SubscriberRepository {
	return &SubscriberRepository{db: db}
}

func (r *SubscriberRepository) ListUserIDsByShop(ctx context.Context, shopID string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT user_id FROM shop_subscriptions WHERE shop_id = $1`, shopID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	userIDs := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		userIDs = append(userIDs, id)
	}
	return userIDs, rows.Err()
}

func (r *SubscriberRepository) ShopName(ctx context.Context, shopID string) (string, error) {
	var name string
	err := r.db.QueryRowContext(ctx, `SELECT name FROM shops WHERE id = $1`, shopID).Scan(&name)
	if err != nil {
		return "", err
	}
	return name, nil
}
