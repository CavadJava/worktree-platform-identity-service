package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"shop-service/internal/models"
)

var (
	ErrCouponNotFound   = errors.New("coupon not found")
	ErrCouponCodeExists = errors.New("coupon code already exists")
)

type CouponRepository struct {
	db *sql.DB
}

func NewCouponRepository(db *sql.DB) *CouponRepository {
	return &CouponRepository{db: db}
}

func (r *CouponRepository) Create(ctx context.Context, c *models.Coupon) error {
	const q = `
		INSERT INTO coupons (id, code, title, discount_percent, valid_until, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.ExecContext(ctx, q, c.ID, c.Code, c.Title, c.DiscountPercent, c.ValidUntil, c.CreatedAt)
	if err != nil && strings.Contains(err.Error(), "duplicate key") {
		return ErrCouponCodeExists
	}
	return err
}

func (r *CouponRepository) FindByID(ctx context.Context, id string) (*models.Coupon, error) {
	const q = `SELECT id, code, title, discount_percent, valid_until, created_at FROM coupons WHERE id = $1`
	c := &models.Coupon{}
	err := r.db.QueryRowContext(ctx, q, id).Scan(&c.ID, &c.Code, &c.Title, &c.DiscountPercent, &c.ValidUntil, &c.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCouponNotFound
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

// ListActive returns coupons that are still valid (no expiry, or expiry in
// the future), newest first.
func (r *CouponRepository) ListActive(ctx context.Context) ([]*models.Coupon, error) {
	const q = `
		SELECT id, code, title, discount_percent, valid_until, created_at
		FROM coupons
		WHERE valid_until IS NULL OR valid_until > now()
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	coupons := []*models.Coupon{}
	for rows.Next() {
		c := &models.Coupon{}
		if err := rows.Scan(&c.ID, &c.Code, &c.Title, &c.DiscountPercent, &c.ValidUntil, &c.CreatedAt); err != nil {
			return nil, err
		}
		coupons = append(coupons, c)
	}
	return coupons, rows.Err()
}

// Claim is idempotent — claiming an already-claimed coupon is a no-op.
func (r *CouponRepository) Claim(ctx context.Context, userID, couponID string) error {
	const q = `
		INSERT INTO user_coupons (user_id, coupon_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, coupon_id) DO NOTHING
	`
	_, err := r.db.ExecContext(ctx, q, userID, couponID)
	return err
}

func (r *CouponRepository) ListClaimed(ctx context.Context, userID string) ([]*models.UserCoupon, error) {
	const q = `
		SELECT c.id, c.code, c.title, c.discount_percent, c.valid_until, c.created_at, uc.claimed_at
		FROM user_coupons uc
		JOIN coupons c ON c.id = uc.coupon_id
		WHERE uc.user_id = $1
		ORDER BY uc.claimed_at DESC
	`
	rows, err := r.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	coupons := []*models.UserCoupon{}
	for rows.Next() {
		c := &models.UserCoupon{}
		if err := rows.Scan(&c.ID, &c.Code, &c.Title, &c.DiscountPercent, &c.ValidUntil, &c.CreatedAt, &c.ClaimedAt); err != nil {
			return nil, err
		}
		coupons = append(coupons, c)
	}
	return coupons, rows.Err()
}

func (r *CouponRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM coupons WHERE id = $1`, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrCouponNotFound
	}
	return nil
}
