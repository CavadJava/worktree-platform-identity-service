package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"

	"registration-service/internal/models"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrEmailTaken   = errors.New("email already registered")
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, u *models.User) error {
	const q = `
		INSERT INTO users (id, email, password_hash, full_name, phone, role, shop_id, shop_role_level, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING user_seq
	`
	err := r.db.QueryRowContext(ctx, q,
		u.ID, u.Email, u.PasswordHash, u.FullName, u.Phone, u.Role, u.ShopID, u.ShopRoleLevel, u.CreatedAt, u.UpdatedAt,
	).Scan(&u.UserSeq)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrEmailTaken
		}
		return err
	}
	return nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	const q = `
		SELECT id, email, password_hash, full_name, COALESCE(phone, ''), role, shop_id, shop_role_level, user_seq, created_at, updated_at
		FROM users WHERE email = $1
	`
	u := &models.User{}
	err := r.db.QueryRowContext(ctx, q, email).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.FullName, &u.Phone, &u.Role, &u.ShopID, &u.ShopRoleLevel, &u.UserSeq, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (*models.User, error) {
	const q = `
		SELECT id, email, password_hash, full_name, COALESCE(phone, ''), role, shop_id, shop_role_level, user_seq, created_at, updated_at
		FROM users WHERE id = $1
	`
	u := &models.User{}
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.FullName, &u.Phone, &u.Role, &u.ShopID, &u.ShopRoleLevel, &u.UserSeq, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *UserRepository) List(ctx context.Context, emailFilter string) ([]*models.User, error) {
	q := `
		SELECT id, email, password_hash, full_name, COALESCE(phone, ''), role, shop_id, shop_role_level, user_seq, created_at, updated_at
		FROM users
	`
	args := []interface{}{}
	if emailFilter != "" {
		q += ` WHERE email ILIKE $1`
		args = append(args, "%"+emailFilter+"%")
	}
	q += ` ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []*models.User{}
	for rows.Next() {
		u := &models.User{}
		if err := rows.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FullName, &u.Phone, &u.Role, &u.ShopID, &u.ShopRoleLevel, &u.UserSeq, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}
