package repository

import (
	"context"
	"database/sql"
	"time"

	"shop-product-service/internal/models"
)

type FavoriteRepository struct {
	db *sql.DB
}

func NewFavoriteRepository(db *sql.DB) *FavoriteRepository {
	return &FavoriteRepository{db: db}
}

func (r *FavoriteRepository) Add(ctx context.Context, userID, productID string) error {
	const q = `
		INSERT INTO product_favorites (user_id, product_id, created_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, product_id) DO NOTHING
	`
	_, err := r.db.ExecContext(ctx, q, userID, productID, time.Now().UTC())
	return err
}

func (r *FavoriteRepository) Remove(ctx context.Context, userID, productID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM product_favorites WHERE user_id = $1 AND product_id = $2`, userID, productID)
	return err
}

// ListByUser returns the full product rows a user has favorited, newest first.
func (r *FavoriteRepository) ListByUser(ctx context.Context, userID string) ([]*models.Product, error) {
	const q = `
		SELECT p.id, p.shop_id, p.name, COALESCE(p.description, ''), p.price, p.stock, p.created_at, p.updated_at
		FROM product_favorites f
		JOIN products p ON p.id = f.product_id
		WHERE f.user_id = $1
		ORDER BY f.created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := []*models.Product{}
	for rows.Next() {
		p := &models.Product{}
		if err := rows.Scan(&p.ID, &p.ShopID, &p.Name, &p.Description, &p.Price, &p.Stock, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, rows.Err()
}
