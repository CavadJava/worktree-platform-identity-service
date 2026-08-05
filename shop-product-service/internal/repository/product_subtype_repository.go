package repository

import (
	"context"
	"database/sql"
	"errors"

	"shop-product-service/internal/models"
)

var ErrProductSubtypeNotFound = errors.New("product subtype not found")

type ProductSubtypeRepository struct {
	db *sql.DB
}

func NewProductSubtypeRepository(db *sql.DB) *ProductSubtypeRepository {
	return &ProductSubtypeRepository{db: db}
}

func (r *ProductSubtypeRepository) Create(ctx context.Context, st *models.ProductSubtype) error {
	const q = `
		INSERT INTO product_subtypes (id, product_type_id, name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.ExecContext(ctx, q, st.ID, st.ProductTypeID, st.Name, st.CreatedAt, st.UpdatedAt)
	return err
}

func (r *ProductSubtypeRepository) FindByID(ctx context.Context, id string) (*models.ProductSubtype, error) {
	const q = `SELECT id, product_type_id, name, created_at, updated_at FROM product_subtypes WHERE id = $1`
	st := &models.ProductSubtype{}
	err := r.db.QueryRowContext(ctx, q, id).Scan(&st.ID, &st.ProductTypeID, &st.Name, &st.CreatedAt, &st.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrProductSubtypeNotFound
	}
	if err != nil {
		return nil, err
	}
	return st, nil
}

func (r *ProductSubtypeRepository) ListByType(ctx context.Context, productTypeID string) ([]*models.ProductSubtype, error) {
	const q = `
		SELECT id, product_type_id, name, created_at, updated_at
		FROM product_subtypes WHERE product_type_id = $1
		ORDER BY created_at ASC
	`
	rows, err := r.db.QueryContext(ctx, q, productTypeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	subtypes := []*models.ProductSubtype{}
	for rows.Next() {
		st := &models.ProductSubtype{}
		if err := rows.Scan(&st.ID, &st.ProductTypeID, &st.Name, &st.CreatedAt, &st.UpdatedAt); err != nil {
			return nil, err
		}
		subtypes = append(subtypes, st)
	}
	return subtypes, rows.Err()
}

func (r *ProductSubtypeRepository) Update(ctx context.Context, id, name string) (*models.ProductSubtype, error) {
	const q = `
		UPDATE product_subtypes SET name = $2, updated_at = now()
		WHERE id = $1
		RETURNING id, product_type_id, name, created_at, updated_at
	`
	st := &models.ProductSubtype{}
	err := r.db.QueryRowContext(ctx, q, id, name).Scan(&st.ID, &st.ProductTypeID, &st.Name, &st.CreatedAt, &st.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrProductSubtypeNotFound
	}
	if err != nil {
		return nil, err
	}
	return st, nil
}

func (r *ProductSubtypeRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM product_subtypes WHERE id = $1`, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrProductSubtypeNotFound
	}
	return nil
}
