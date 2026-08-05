package repository

import (
	"context"
	"database/sql"
	"errors"

	"shop-product-service/internal/models"
)

var ErrProductNotFound = errors.New("product not found")

type ProductRepository struct {
	db *sql.DB
}

func NewProductRepository(db *sql.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

func (r *ProductRepository) Create(ctx context.Context, p *models.Product) error {
	const q = `
		INSERT INTO products (id, shop_id, name, description, price, stock, product_type_id, brand, material, weight_kg, origin_country, warranty_months, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`
	_, err := r.db.ExecContext(ctx, q,
		p.ID, p.ShopID, p.Name, p.Description, p.Price, p.Stock, p.ProductTypeID,
		p.Brand, p.Material, p.WeightKg, p.OriginCountry, p.WarrantyMonths,
		p.CreatedAt, p.UpdatedAt,
	)
	return err
}

func (r *ProductRepository) FindByID(ctx context.Context, id string) (*models.Product, error) {
	const q = `
		SELECT id, shop_id, name, COALESCE(description, ''), price, stock, product_type_id,
		       brand, material, weight_kg, origin_country, warranty_months, created_at, updated_at
		FROM products WHERE id = $1
	`
	p := &models.Product{}
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&p.ID, &p.ShopID, &p.Name, &p.Description, &p.Price, &p.Stock, &p.ProductTypeID,
		&p.Brand, &p.Material, &p.WeightKg, &p.OriginCountry, &p.WarrantyMonths,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrProductNotFound
	}
	if err != nil {
		return nil, err
	}
	p.ComputeFlags()
	return p, nil
}

func (r *ProductRepository) List(ctx context.Context, shopID string) ([]*models.Product, error) {
	q := `
		SELECT id, shop_id, name, COALESCE(description, ''), price, stock, product_type_id,
		       brand, material, weight_kg, origin_country, warranty_months, created_at, updated_at
		FROM products
	`
	args := []interface{}{}
	if shopID != "" {
		q += ` WHERE shop_id = $1`
		args = append(args, shopID)
	}
	q += ` ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := []*models.Product{}
	for rows.Next() {
		p := &models.Product{}
		if err := rows.Scan(
			&p.ID, &p.ShopID, &p.Name, &p.Description, &p.Price, &p.Stock, &p.ProductTypeID,
			&p.Brand, &p.Material, &p.WeightKg, &p.OriginCountry, &p.WarrantyMonths,
			&p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, err
		}
		p.ComputeFlags()
		products = append(products, p)
	}
	return products, rows.Err()
}

func (r *ProductRepository) Update(ctx context.Context, id, name, description string, price float64, stock int, productTypeID *string, details models.ProductDetails) (*models.Product, error) {
	const q = `
		UPDATE products
		SET name = $2, description = $3, price = $4, stock = $5, product_type_id = $6,
		    brand = $7, material = $8, weight_kg = $9, origin_country = $10, warranty_months = $11,
		    updated_at = now()
		WHERE id = $1
		RETURNING id, shop_id, name, COALESCE(description, ''), price, stock, product_type_id,
		          brand, material, weight_kg, origin_country, warranty_months, created_at, updated_at
	`
	p := &models.Product{}
	err := r.db.QueryRowContext(ctx, q,
		id, name, description, price, stock, productTypeID,
		details.Brand, details.Material, details.WeightKg, details.OriginCountry, details.WarrantyMonths,
	).Scan(
		&p.ID, &p.ShopID, &p.Name, &p.Description, &p.Price, &p.Stock, &p.ProductTypeID,
		&p.Brand, &p.Material, &p.WeightKg, &p.OriginCountry, &p.WarrantyMonths,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrProductNotFound
	}
	if err != nil {
		return nil, err
	}
	p.ComputeFlags()
	return p, nil
}

func (r *ProductRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM products WHERE id = $1`, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrProductNotFound
	}
	return nil
}
