package repository

import (
	"context"
	"database/sql"

	"platform-identity-service/internal/models"
)

type ShopRoleRepository struct {
	db *sql.DB
}

func NewShopRoleRepository(db *sql.DB) *ShopRoleRepository {
	return &ShopRoleRepository{db: db}
}

func (r *ShopRoleRepository) List(ctx context.Context) ([]models.ShopRole, error) {
	const q = `SELECT id, name FROM shop_roles ORDER BY id`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	roles := []models.ShopRole{}
	for rows.Next() {
		var role models.ShopRole
		if err := rows.Scan(&role.ID, &role.Name); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, rows.Err()
}
