package repository

import (
	"context"
	"database/sql"
	"errors"

	"shop-product-service/internal/models"
)

var ErrProductTypeNotFound = errors.New("product type not found")

type ProductTypeRepository struct {
	db *sql.DB
}

func NewProductTypeRepository(db *sql.DB) *ProductTypeRepository {
	return &ProductTypeRepository{db: db}
}

func (r *ProductTypeRepository) Create(ctx context.Context, t *models.ProductType) error {
	const q = `
		INSERT INTO product_types (id, shop_id, name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.ExecContext(ctx, q, t.ID, t.ShopID, t.Name, t.CreatedAt, t.UpdatedAt)
	return err
}

func (r *ProductTypeRepository) FindByID(ctx context.Context, id string) (*models.ProductType, error) {
	const q = `SELECT id, shop_id, name, created_at, updated_at FROM product_types WHERE id = $1`
	t := &models.ProductType{}
	err := r.db.QueryRowContext(ctx, q, id).Scan(&t.ID, &t.ShopID, &t.Name, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrProductTypeNotFound
	}
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (r *ProductTypeRepository) ListByShop(ctx context.Context, shopID string) ([]*models.ProductType, error) {
	const q = `
		SELECT id, shop_id, name, created_at, updated_at
		FROM product_types WHERE shop_id = $1
		ORDER BY created_at ASC
	`
	rows, err := r.db.QueryContext(ctx, q, shopID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	types := []*models.ProductType{}
	for rows.Next() {
		t := &models.ProductType{}
		if err := rows.Scan(&t.ID, &t.ShopID, &t.Name, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		types = append(types, t)
	}
	return types, rows.Err()
}

func (r *ProductTypeRepository) Update(ctx context.Context, id, name string) (*models.ProductType, error) {
	const q = `
		UPDATE product_types SET name = $2, updated_at = now()
		WHERE id = $1
		RETURNING id, shop_id, name, created_at, updated_at
	`
	t := &models.ProductType{}
	err := r.db.QueryRowContext(ctx, q, id, name).Scan(&t.ID, &t.ShopID, &t.Name, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrProductTypeNotFound
	}
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (r *ProductTypeRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM product_types WHERE id = $1`, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrProductTypeNotFound
	}
	return nil
}
