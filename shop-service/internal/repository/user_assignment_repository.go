package repository

import (
	"context"
	"database/sql"
)

// UserAssignmentRepository writes to / reads from the `users` table that
// registration-service owns the schema for. It's used by the shop
// application flow: to grant/revoke `shop_id`/`shop_role_level` on
// send-form/approve/reject, and to look up an applicant's contact details
// so shop-service can notify them of application status changes.
type UserAssignmentRepository struct {
	db *sql.DB
}

func NewUserAssignmentRepository(db *sql.DB) *UserAssignmentRepository {
	return &UserAssignmentRepository{db: db}
}

func (r *UserAssignmentRepository) AssignShopAdmin(ctx context.Context, userID, shopID string, level int) error {
	const q = `UPDATE users SET shop_id = $2, shop_role_level = $3, updated_at = now() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, q, userID, shopID, level)
	return err
}

// RevokeShopAssignment clears a user's shop membership — used when a
// temporary shop is rejected at the form_sent stage.
func (r *UserAssignmentRepository) RevokeShopAssignment(ctx context.Context, userID string) error {
	const q = `UPDATE users SET shop_id = NULL, shop_role_level = 0, updated_at = now() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, q, userID)
	return err
}

type Contact struct {
	Email    string
	FullName string
}

func (r *UserAssignmentRepository) GetContact(ctx context.Context, userID string) (*Contact, error) {
	const q = `SELECT email, full_name FROM users WHERE id = $1`
	c := &Contact{}
	if err := r.db.QueryRowContext(ctx, q, userID).Scan(&c.Email, &c.FullName); err != nil {
		return nil, err
	}
	return c, nil
}
