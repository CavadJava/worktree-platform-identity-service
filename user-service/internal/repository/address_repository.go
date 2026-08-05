package repository

import (
	"context"
	"database/sql"
	"errors"

	"user-service/internal/models"
)

var ErrAddressNotFound = errors.New("address not found")

type AddressRepository struct {
	db *sql.DB
}

func NewAddressRepository(db *sql.DB) *AddressRepository {
	return &AddressRepository{db: db}
}

func (r *AddressRepository) Create(ctx context.Context, a *models.Address) error {
	const q = `
		INSERT INTO user_addresses (id, user_id, title, full_address, city, phone, is_default, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''), NULLIF($6, ''), $7, $8, $9)
	`
	_, err := r.db.ExecContext(ctx, q, a.ID, a.UserID, a.Title, a.FullAddress, a.City, a.Phone, a.IsDefault, a.CreatedAt, a.UpdatedAt)
	return err
}

func (r *AddressRepository) FindByID(ctx context.Context, id string) (*models.Address, error) {
	const q = `
		SELECT id, user_id, title, full_address, COALESCE(city, ''), COALESCE(phone, ''), is_default, created_at, updated_at
		FROM user_addresses WHERE id = $1
	`
	a := &models.Address{}
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&a.ID, &a.UserID, &a.Title, &a.FullAddress, &a.City, &a.Phone, &a.IsDefault, &a.CreatedAt, &a.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrAddressNotFound
	}
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (r *AddressRepository) ListByUser(ctx context.Context, userID string) ([]*models.Address, error) {
	const q = `
		SELECT id, user_id, title, full_address, COALESCE(city, ''), COALESCE(phone, ''), is_default, created_at, updated_at
		FROM user_addresses WHERE user_id = $1
		ORDER BY is_default DESC, created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	addresses := []*models.Address{}
	for rows.Next() {
		a := &models.Address{}
		if err := rows.Scan(&a.ID, &a.UserID, &a.Title, &a.FullAddress, &a.City, &a.Phone, &a.IsDefault, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		addresses = append(addresses, a)
	}
	return addresses, rows.Err()
}

func (r *AddressRepository) Update(ctx context.Context, id, title, fullAddress, city, phone string, isDefault bool) (*models.Address, error) {
	const q = `
		UPDATE user_addresses
		SET title = $2, full_address = $3, city = NULLIF($4, ''), phone = NULLIF($5, ''), is_default = $6, updated_at = now()
		WHERE id = $1
		RETURNING id, user_id, title, full_address, COALESCE(city, ''), COALESCE(phone, ''), is_default, created_at, updated_at
	`
	a := &models.Address{}
	err := r.db.QueryRowContext(ctx, q, id, title, fullAddress, city, phone, isDefault).Scan(
		&a.ID, &a.UserID, &a.Title, &a.FullAddress, &a.City, &a.Phone, &a.IsDefault, &a.CreatedAt, &a.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrAddressNotFound
	}
	if err != nil {
		return nil, err
	}
	return a, nil
}

// ClearDefault removes the default flag from every address of the user —
// called before setting a new default so at most one stays default.
func (r *AddressRepository) ClearDefault(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE user_addresses SET is_default = false WHERE user_id = $1`, userID)
	return err
}

func (r *AddressRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM user_addresses WHERE id = $1`, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrAddressNotFound
	}
	return nil
}
