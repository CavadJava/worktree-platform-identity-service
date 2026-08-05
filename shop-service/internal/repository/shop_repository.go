package repository

import (
	"context"
	"database/sql"
	"errors"

	"shop-service/internal/models"
)

var ErrShopNotFound = errors.New("shop not found")

type ShopRepository struct {
	db *sql.DB
}

func NewShopRepository(db *sql.DB) *ShopRepository {
	return &ShopRepository{db: db}
}

func (r *ShopRepository) Create(ctx context.Context, s *models.Shop) error {
	const q = `
		INSERT INTO shops (id, owner_id, name, description, temporary, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING shop_seq
	`
	return r.db.QueryRowContext(ctx, q, s.ID, s.OwnerID, s.Name, s.Description, s.Temporary, s.CreatedAt, s.UpdatedAt).Scan(&s.ShopSeq)
}

func (r *ShopRepository) FindByID(ctx context.Context, id string) (*models.Shop, error) {
	const q = `
		SELECT id, owner_id, name, COALESCE(description, ''), temporary, shop_seq, created_at, updated_at
		FROM shops WHERE id = $1
	`
	s := &models.Shop{}
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&s.ID, &s.OwnerID, &s.Name, &s.Description, &s.Temporary, &s.ShopSeq, &s.CreatedAt, &s.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrShopNotFound
	}
	if err != nil {
		return nil, err
	}
	return s, nil
}

// List returns shops for public browsing — temporary (not-yet-approved)
// shops never appear here, since their listings/products aren't vetted yet.
func (r *ShopRepository) List(ctx context.Context, ownerID string) ([]*models.Shop, error) {
	q := `
		SELECT id, owner_id, name, COALESCE(description, ''), temporary, shop_seq, created_at, updated_at
		FROM shops
		WHERE temporary = false
	`
	args := []interface{}{}
	if ownerID != "" {
		q += ` AND owner_id = $1`
		args = append(args, ownerID)
	}
	q += ` ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	shops := []*models.Shop{}
	for rows.Next() {
		s := &models.Shop{}
		if err := rows.Scan(&s.ID, &s.OwnerID, &s.Name, &s.Description, &s.Temporary, &s.ShopSeq, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		shops = append(shops, s)
	}
	return shops, rows.Err()
}

func (r *ShopRepository) Update(ctx context.Context, id, name, description string) (*models.Shop, error) {
	const q = `
		UPDATE shops SET name = $2, description = $3, updated_at = now()
		WHERE id = $1
		RETURNING id, owner_id, name, COALESCE(description, ''), temporary, shop_seq, created_at, updated_at
	`
	s := &models.Shop{}
	err := r.db.QueryRowContext(ctx, q, id, name, description).Scan(
		&s.ID, &s.OwnerID, &s.Name, &s.Description, &s.Temporary, &s.ShopSeq, &s.CreatedAt, &s.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrShopNotFound
	}
	if err != nil {
		return nil, err
	}
	return s, nil
}

// SetTemporary flips a shop between temporary (pending final review) and
// permanent — called when a shop application is approved.
func (r *ShopRepository) SetTemporary(ctx context.Context, id string, temporary bool) error {
	result, err := r.db.ExecContext(ctx, `UPDATE shops SET temporary = $2, updated_at = now() WHERE id = $1`, id, temporary)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrShopNotFound
	}
	return nil
}

func (r *ShopRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM shops WHERE id = $1`, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrShopNotFound
	}
	return nil
}
