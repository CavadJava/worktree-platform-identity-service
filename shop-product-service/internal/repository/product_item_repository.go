package repository

import (
	"context"
	"database/sql"
	"errors"

	"shop-product-service/internal/models"
)

var ErrProductItemNotFound = errors.New("product item not found")

type ProductItemRepository struct {
	db *sql.DB
}

func NewProductItemRepository(db *sql.DB) *ProductItemRepository {
	return &ProductItemRepository{db: db}
}

func (r *ProductItemRepository) Create(ctx context.Context, i *models.ProductItem) error {
	const q = `
		INSERT INTO product_items (id, product_id, name, price, stock, is_discounted, discount_price, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.ExecContext(ctx, q,
		i.ID, i.ProductID, i.Name, i.Price, i.Stock, i.IsDiscounted, i.DiscountPrice, i.CreatedAt, i.UpdatedAt,
	)
	return err
}

func (r *ProductItemRepository) FindByID(ctx context.Context, id string) (*models.ProductItem, error) {
	const q = `
		SELECT id, product_id, name, price, stock, is_discounted, discount_price, created_at, updated_at
		FROM product_items WHERE id = $1
	`
	i := &models.ProductItem{}
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&i.ID, &i.ProductID, &i.Name, &i.Price, &i.Stock, &i.IsDiscounted, &i.DiscountPrice, &i.CreatedAt, &i.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrProductItemNotFound
	}
	if err != nil {
		return nil, err
	}
	return i, nil
}

func (r *ProductItemRepository) ListByProduct(ctx context.Context, productID string) ([]*models.ProductItem, error) {
	const q = `
		SELECT id, product_id, name, price, stock, is_discounted, discount_price, created_at, updated_at
		FROM product_items WHERE product_id = $1
		ORDER BY created_at ASC
	`
	rows, err := r.db.QueryContext(ctx, q, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []*models.ProductItem{}
	for rows.Next() {
		i := &models.ProductItem{}
		if err := rows.Scan(&i.ID, &i.ProductID, &i.Name, &i.Price, &i.Stock, &i.IsDiscounted, &i.DiscountPrice, &i.CreatedAt, &i.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

func (r *ProductItemRepository) Update(ctx context.Context, id, name string, price float64, stock int, isDiscounted bool, discountPrice *float64) (*models.ProductItem, error) {
	const q = `
		UPDATE product_items
		SET name = $2, price = $3, stock = $4, is_discounted = $5, discount_price = $6, updated_at = now()
		WHERE id = $1
		RETURNING id, product_id, name, price, stock, is_discounted, discount_price, created_at, updated_at
	`
	i := &models.ProductItem{}
	err := r.db.QueryRowContext(ctx, q, id, name, price, stock, isDiscounted, discountPrice).Scan(
		&i.ID, &i.ProductID, &i.Name, &i.Price, &i.Stock, &i.IsDiscounted, &i.DiscountPrice, &i.CreatedAt, &i.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrProductItemNotFound
	}
	if err != nil {
		return nil, err
	}
	return i, nil
}

func (r *ProductItemRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM product_items WHERE id = $1`, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrProductItemNotFound
	}
	return nil
}
