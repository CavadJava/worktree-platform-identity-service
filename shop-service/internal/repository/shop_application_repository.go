package repository

import (
	"context"
	"database/sql"
	"errors"

	"shop-service/internal/models"
)

var ErrApplicationNotFound = errors.New("shop application not found")

type ShopApplicationRepository struct {
	db *sql.DB
}

func NewShopApplicationRepository(db *sql.DB) *ShopApplicationRepository {
	return &ShopApplicationRepository{db: db}
}

func (r *ShopApplicationRepository) Create(ctx context.Context, a *models.ShopApplication) error {
	const q = `
		INSERT INTO shop_applications (id, applicant_id, name, description, template_version, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.ExecContext(ctx, q,
		a.ID, a.ApplicantID, a.Name, a.Description, a.TemplateVersion, a.Status, a.CreatedAt, a.UpdatedAt,
	)
	return err
}

func (r *ShopApplicationRepository) FindByID(ctx context.Context, id string) (*models.ShopApplication, error) {
	const q = `
		SELECT id, applicant_id, name, COALESCE(description, ''), template_version, status,
		       reviewer_id, COALESCE(review_note, ''), shop_id, created_at, updated_at
		FROM shop_applications WHERE id = $1
	`
	a := &models.ShopApplication{}
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&a.ID, &a.ApplicantID, &a.Name, &a.Description, &a.TemplateVersion, &a.Status,
		&a.ReviewerID, &a.ReviewNote, &a.ShopID, &a.CreatedAt, &a.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrApplicationNotFound
	}
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (r *ShopApplicationRepository) List(ctx context.Context, status string) ([]*models.ShopApplication, error) {
	q := `
		SELECT id, applicant_id, name, COALESCE(description, ''), template_version, status,
		       reviewer_id, COALESCE(review_note, ''), shop_id, created_at, updated_at
		FROM shop_applications
	`
	args := []interface{}{}
	if status != "" {
		q += ` WHERE status = $1`
		args = append(args, status)
	}
	q += ` ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applications := []*models.ShopApplication{}
	for rows.Next() {
		a := &models.ShopApplication{}
		if err := rows.Scan(
			&a.ID, &a.ApplicantID, &a.Name, &a.Description, &a.TemplateVersion, &a.Status,
			&a.ReviewerID, &a.ReviewNote, &a.ShopID, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, err
		}
		applications = append(applications, a)
	}
	return applications, rows.Err()
}

// Decide moves a pending application to approved/rejected. shopID is set
// only on approval (nil on rejection).
func (r *ShopApplicationRepository) Decide(ctx context.Context, id, status, reviewerID, reviewNote string, shopID *string) (*models.ShopApplication, error) {
	const q = `
		UPDATE shop_applications
		SET status = $2, reviewer_id = $3, review_note = $4, shop_id = $5, updated_at = now()
		WHERE id = $1
		RETURNING id, applicant_id, name, COALESCE(description, ''), template_version, status,
		          reviewer_id, COALESCE(review_note, ''), shop_id, created_at, updated_at
	`
	a := &models.ShopApplication{}
	err := r.db.QueryRowContext(ctx, q, id, status, reviewerID, reviewNote, shopID).Scan(
		&a.ID, &a.ApplicantID, &a.Name, &a.Description, &a.TemplateVersion, &a.Status,
		&a.ReviewerID, &a.ReviewNote, &a.ShopID, &a.CreatedAt, &a.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrApplicationNotFound
	}
	if err != nil {
		return nil, err
	}
	return a, nil
}
