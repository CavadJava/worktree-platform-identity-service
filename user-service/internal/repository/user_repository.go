package repository

import (
	"context"
	"database/sql"
	"errors"

	"user-service/internal/models"
)

var ErrUserNotFound = errors.New("user not found")

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (*models.User, error) {
	const q = `
		SELECT id, email, full_name, COALESCE(phone, ''), created_at, updated_at
		FROM users WHERE id = $1
	`
	u := &models.User{}
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&u.ID, &u.Email, &u.FullName, &u.Phone, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *UserRepository) UpdateProfile(ctx context.Context, id, fullName, phone string) (*models.User, error) {
	const q = `
		UPDATE users SET full_name = $2, phone = $3, updated_at = now()
		WHERE id = $1
		RETURNING id, email, full_name, COALESCE(phone, ''), created_at, updated_at
	`
	u := &models.User{}
	err := r.db.QueryRowContext(ctx, q, id, fullName, phone).Scan(
		&u.ID, &u.Email, &u.FullName, &u.Phone, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}
