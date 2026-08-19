package repository

import (
	"context"
	"database/sql"

	"platform-identity-service/internal/models"
)

type SystemRoleRepository struct {
	db *sql.DB
}

func NewSystemRoleRepository(db *sql.DB) *SystemRoleRepository {
	return &SystemRoleRepository{db: db}
}

func (r *SystemRoleRepository) List(ctx context.Context) ([]models.SystemRole, error) {
	const q = `SELECT id, name FROM system_roles ORDER BY id`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	roles := []models.SystemRole{}
	for rows.Next() {
		var role models.SystemRole
		if err := rows.Scan(&role.ID, &role.Name); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, rows.Err()
}
