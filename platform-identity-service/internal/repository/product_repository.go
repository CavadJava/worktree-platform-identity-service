package repository

import (
	"context"
	"database/sql"
	"errors"

	"platform-identity-service/internal/models"
)

var ErrProductNotFound = errors.New("product not found")

type ProductRepository struct {
	db *sql.DB
}

func NewProductRepository(db *sql.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

func (r *ProductRepository) Create(ctx context.Context, p *models.Product) error {
	const q = `INSERT INTO products (id, name, created_at) VALUES ($1, $2, $3)`
	_, err := r.db.ExecContext(ctx, q, p.ID, p.Name, p.CreatedAt)
	return err
}

func (r *ProductRepository) List(ctx context.Context) ([]models.Product, error) {
	const q = `SELECT id, name, description, tech_stack, created_at FROM products ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := []models.Product{}
	for rows.Next() {
		var p models.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.TechStack, &p.CreatedAt); err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, rows.Err()
}

func (r *ProductRepository) GetByID(ctx context.Context, id string) (*models.Product, error) {
	const q = `SELECT id, name, description, tech_stack, created_at FROM products WHERE id = $1`
	var p models.Product
	err := r.db.QueryRowContext(ctx, q, id).Scan(&p.ID, &p.Name, &p.Description, &p.TechStack, &p.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrProductNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// Update sets a product's profile fields. Both fields are always submitted
// together by the admin panel's profile form, so there is no partial-update
// variant here (unlike UserUpdate's nil-field pattern).
func (r *ProductRepository) Update(ctx context.Context, productID, description, techStack string) error {
	const q = `UPDATE products SET description = $2, tech_stack = $3 WHERE id = $1`
	result, err := r.db.ExecContext(ctx, q, productID, description, techStack)
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
