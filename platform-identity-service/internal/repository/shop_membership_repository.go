package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"

	"platform-identity-service/internal/models"
)

var (
	ErrMembershipNotFound = errors.New("shop membership not found")
	ErrMembershipExists   = errors.New("user is already a member of this shop")
)

type ShopMembershipRepository struct {
	db *sql.DB
}

func NewShopMembershipRepository(db *sql.DB) *ShopMembershipRepository {
	return &ShopMembershipRepository{db: db}
}

const selectMembershipWithNames = `
	SELECT m.id, m.user_id, m.shop_id, s.name, m.shop_role_id, sr.name, m.created_at
	FROM user_shop_memberships m
	JOIN shops s ON s.id = m.shop_id
	JOIN shop_roles sr ON sr.id = m.shop_role_id
`

func (r *ShopMembershipRepository) scanOne(row *sql.Row) (*models.ShopMembership, error) {
	var m models.ShopMembership
	err := row.Scan(&m.ID, &m.UserID, &m.ShopID, &m.ShopName, &m.ShopRoleID, &m.ShopRoleName, &m.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrMembershipNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *ShopMembershipRepository) Create(ctx context.Context, m *models.ShopMembership) error {
	const q = `
		INSERT INTO user_shop_memberships (id, user_id, shop_id, shop_role_id, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.ExecContext(ctx, q, m.ID, m.UserID, m.ShopID, m.ShopRoleID, m.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrMembershipExists
		}
		return err
	}
	return nil
}

func (r *ShopMembershipRepository) GetByUserAndShop(ctx context.Context, userID, shopID string) (*models.ShopMembership, error) {
	row := r.db.QueryRowContext(ctx, selectMembershipWithNames+" WHERE m.user_id = $1 AND m.shop_id = $2", userID, shopID)
	return r.scanOne(row)
}

func (r *ShopMembershipRepository) ListByShop(ctx context.Context, shopID string) ([]models.ShopMembership, error) {
	rows, err := r.db.QueryContext(ctx, selectMembershipWithNames+" WHERE m.shop_id = $1 ORDER BY m.created_at", shopID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	memberships := []models.ShopMembership{}
	for rows.Next() {
		var m models.ShopMembership
		if err := rows.Scan(&m.ID, &m.UserID, &m.ShopID, &m.ShopName, &m.ShopRoleID, &m.ShopRoleName, &m.CreatedAt); err != nil {
			return nil, err
		}
		memberships = append(memberships, m)
	}
	return memberships, rows.Err()
}

func (r *ShopMembershipRepository) ListByUser(ctx context.Context, userID string) ([]models.ShopMembership, error) {
	rows, err := r.db.QueryContext(ctx, selectMembershipWithNames+" WHERE m.user_id = $1 ORDER BY m.created_at", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	memberships := []models.ShopMembership{}
	for rows.Next() {
		var m models.ShopMembership
		if err := rows.Scan(&m.ID, &m.UserID, &m.ShopID, &m.ShopName, &m.ShopRoleID, &m.ShopRoleName, &m.CreatedAt); err != nil {
			return nil, err
		}
		memberships = append(memberships, m)
	}
	return memberships, rows.Err()
}

func (r *ShopMembershipRepository) Delete(ctx context.Context, userID, shopID string) error {
	const q = `DELETE FROM user_shop_memberships WHERE user_id = $1 AND shop_id = $2`
	result, err := r.db.ExecContext(ctx, q, userID, shopID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrMembershipNotFound
	}
	return nil
}

func (r *ShopMembershipRepository) SetShopRole(ctx context.Context, userID, shopID string, shopRoleID int16) error {
	const q = `UPDATE user_shop_memberships SET shop_role_id = $3 WHERE user_id = $1 AND shop_id = $2`
	result, err := r.db.ExecContext(ctx, q, userID, shopID, shopRoleID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrMembershipNotFound
	}
	return nil
}
