package repository

import (
	"context"
	"database/sql"
	"errors"
)

var ErrUserNotFound = errors.New("user not found")

type RoleRepository struct {
	db *sql.DB
}

func NewRoleRepository(db *sql.DB) *RoleRepository {
	return &RoleRepository{db: db}
}

func (r *RoleRepository) GetAssignment(ctx context.Context, userID string) (*string, int, error) {
	const q = `SELECT shop_id, shop_role_level FROM users WHERE id = $1`
	var shopID *string
	var level int
	err := r.db.QueryRowContext(ctx, q, userID).Scan(&shopID, &level)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, 0, ErrUserNotFound
	}
	if err != nil {
		return nil, 0, err
	}
	return shopID, level, nil
}

func (r *RoleRepository) SetAssignment(ctx context.Context, userID string, shopID *string, level int) error {
	const q = `UPDATE users SET shop_id = $2, shop_role_level = $3, updated_at = now() WHERE id = $1`
	result, err := r.db.ExecContext(ctx, q, userID, shopID, level)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrUserNotFound
	}
	return nil
}

type StaffMember struct {
	UserID        string
	Email         string
	FullName      string
	ShopRoleLevel int
}

func (r *RoleRepository) ListByShop(ctx context.Context, shopID string) ([]StaffMember, error) {
	const q = `
		SELECT id, email, full_name, shop_role_level
		FROM users
		WHERE shop_id = $1
		ORDER BY shop_role_level DESC, email
	`
	rows, err := r.db.QueryContext(ctx, q, shopID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	staff := []StaffMember{}
	for rows.Next() {
		var m StaffMember
		if err := rows.Scan(&m.UserID, &m.Email, &m.FullName, &m.ShopRoleLevel); err != nil {
			return nil, err
		}
		staff = append(staff, m)
	}
	return staff, rows.Err()
}
