package repository

import (
	"context"
	"database/sql"
	"errors"

	"platform-identity-service/internal/models"
)

var ErrSubprojectNotFound = errors.New("subproject not found")

type SubprojectRepository struct {
	db *sql.DB
}

func NewSubprojectRepository(db *sql.DB) *SubprojectRepository {
	return &SubprojectRepository{db: db}
}

func (r *SubprojectRepository) Create(ctx context.Context, s *models.ProductSubproject) error {
	const q = `
		INSERT INTO product_subprojects (id, product_id, name, description, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.ExecContext(ctx, q, s.ID, s.ProductID, s.Name, s.Description, s.CreatedAt)
	return err
}

func (r *SubprojectRepository) GetByID(ctx context.Context, id string) (*models.ProductSubproject, error) {
	const q = `SELECT id, product_id, name, description, created_at FROM product_subprojects WHERE id = $1`
	var s models.ProductSubproject
	err := r.db.QueryRowContext(ctx, q, id).Scan(&s.ID, &s.ProductID, &s.Name, &s.Description, &s.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrSubprojectNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SubprojectRepository) ListByProduct(ctx context.Context, productID string) ([]models.ProductSubproject, error) {
	const q = `
		SELECT id, product_id, name, description, created_at
		FROM product_subprojects
		WHERE product_id = $1
		ORDER BY created_at
	`
	rows, err := r.db.QueryContext(ctx, q, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	subprojects := []models.ProductSubproject{}
	for rows.Next() {
		var s models.ProductSubproject
		if err := rows.Scan(&s.ID, &s.ProductID, &s.Name, &s.Description, &s.CreatedAt); err != nil {
			return nil, err
		}
		subprojects = append(subprojects, s)
	}
	return subprojects, rows.Err()
}

func (r *SubprojectRepository) Delete(ctx context.Context, id string) error {
	const q = `DELETE FROM product_subprojects WHERE id = $1`
	result, err := r.db.ExecContext(ctx, q, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrSubprojectNotFound
	}
	return nil
}
