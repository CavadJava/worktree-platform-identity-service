package repository

import (
	"context"
	"database/sql"
	"errors"

	"platform-identity-service/internal/models"
)

var ErrProductAdminRequestNotFound = errors.New("product admin request not found")

type ProductAdminRequestRepository struct {
	db *sql.DB
}

func NewProductAdminRequestRepository(db *sql.DB) *ProductAdminRequestRepository {
	return &ProductAdminRequestRepository{db: db}
}

func (r *ProductAdminRequestRepository) Create(ctx context.Context, req *models.ProductAdminRequest) error {
	const q = `
		INSERT INTO product_admin_requests (id, product_id, subject_user_id, requested_by_user_id, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.ExecContext(ctx, q, req.ID, req.ProductID, req.SubjectUserID, req.RequestedByUserID, req.Status, req.CreatedAt)
	return err
}

const selectProductAdminRequest = `
	SELECT id, product_id, subject_user_id, requested_by_user_id, status, created_at, decided_at, decided_by_user_id
	FROM product_admin_requests
`

func scanProductAdminRequest(row *sql.Row) (*models.ProductAdminRequest, error) {
	var req models.ProductAdminRequest
	err := row.Scan(&req.ID, &req.ProductID, &req.SubjectUserID, &req.RequestedByUserID, &req.Status, &req.CreatedAt, &req.DecidedAt, &req.DecidedByUserID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrProductAdminRequestNotFound
	}
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func (r *ProductAdminRequestRepository) GetByID(ctx context.Context, id string) (*models.ProductAdminRequest, error) {
	row := r.db.QueryRowContext(ctx, selectProductAdminRequest+" WHERE id = $1", id)
	return scanProductAdminRequest(row)
}

func (r *ProductAdminRequestRepository) ListPendingByProduct(ctx context.Context, productID string) ([]models.ProductAdminRequest, error) {
	rows, err := r.db.QueryContext(ctx, selectProductAdminRequest+" WHERE product_id = $1 AND status = 'pending' ORDER BY created_at", productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	requests := []models.ProductAdminRequest{}
	for rows.Next() {
		var req models.ProductAdminRequest
		if err := rows.Scan(&req.ID, &req.ProductID, &req.SubjectUserID, &req.RequestedByUserID, &req.Status, &req.CreatedAt, &req.DecidedAt, &req.DecidedByUserID); err != nil {
			return nil, err
		}
		requests = append(requests, req)
	}
	return requests, rows.Err()
}

func (r *ProductAdminRequestRepository) HasPending(ctx context.Context, productID, subjectUserID string) (bool, error) {
	const q = `SELECT EXISTS(SELECT 1 FROM product_admin_requests WHERE product_id = $1 AND subject_user_id = $2 AND status = 'pending')`
	var exists bool
	err := r.db.QueryRowContext(ctx, q, productID, subjectUserID).Scan(&exists)
	return exists, err
}

func (r *ProductAdminRequestRepository) SetStatus(ctx context.Context, id, status, decidedByUserID string) error {
	const q = `UPDATE product_admin_requests SET status = $2, decided_at = now(), decided_by_user_id = $3 WHERE id = $1`
	result, err := r.db.ExecContext(ctx, q, id, status, decidedByUserID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrProductAdminRequestNotFound
	}
	return nil
}
