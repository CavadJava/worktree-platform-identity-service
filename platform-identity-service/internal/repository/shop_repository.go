package repository

import (
	"context"
	"database/sql"
	"errors"

	"platform-identity-service/internal/models"
)

var ErrShopNotFound = errors.New("shop not found")

type ShopRepository struct {
	db *sql.DB
}

func NewShopRepository(db *sql.DB) *ShopRepository {
	return &ShopRepository{db: db}
}

func (r *ShopRepository) Create(ctx context.Context, s *models.Shop) error {
	const q = `INSERT INTO shops (id, name, created_at) VALUES ($1, $2, $3)`
	_, err := r.db.ExecContext(ctx, q, s.ID, s.Name, s.CreatedAt)
	return err
}

func (r *ShopRepository) List(ctx context.Context) ([]models.Shop, error) {
	const q = `SELECT id, name, created_at FROM shops ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	shops := []models.Shop{}
	for rows.Next() {
		var s models.Shop
		if err := rows.Scan(&s.ID, &s.Name, &s.CreatedAt); err != nil {
			return nil, err
		}
		shops = append(shops, s)
	}
	return shops, rows.Err()
}

func (r *ShopRepository) GetByID(ctx context.Context, id string) (*models.Shop, error) {
	const q = `SELECT id, name, created_at FROM shops WHERE id = $1`
	var s models.Shop
	err := r.db.QueryRowContext(ctx, q, id).Scan(&s.ID, &s.Name, &s.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrShopNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}
