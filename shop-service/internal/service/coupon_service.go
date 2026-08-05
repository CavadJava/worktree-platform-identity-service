package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"shop-service/internal/models"
	"shop-service/internal/repository"
	"shop-service/internal/roles"
)

var (
	ErrCouponNotFound   = repository.ErrCouponNotFound
	ErrCouponCodeExists = repository.ErrCouponCodeExists
	ErrCouponExpired    = errors.New("coupon has expired")
	ErrInvalidDiscount  = errors.New("discount_percent must be between 1 and 100")
)

type CouponService struct {
	repo *repository.CouponRepository
}

func NewCouponService(repo *repository.CouponRepository) *CouponService {
	return &CouponService{repo: repo}
}

// Create is administrator-only.
func (s *CouponService) Create(ctx context.Context, identity Identity, code, title string, discountPercent int, validUntil *time.Time) (*models.Coupon, error) {
	if identity.Role != roles.RoleAdministrator {
		return nil, ErrForbidden
	}
	if discountPercent < 1 || discountPercent > 100 {
		return nil, ErrInvalidDiscount
	}

	c := &models.Coupon{
		ID:              uuid.NewString(),
		Code:            strings.ToUpper(strings.TrimSpace(code)),
		Title:           strings.TrimSpace(title),
		DiscountPercent: discountPercent,
		ValidUntil:      validUntil,
		CreatedAt:       time.Now().UTC(),
	}
	if err := s.repo.Create(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *CouponService) ListActive(ctx context.Context) ([]*models.Coupon, error) {
	return s.repo.ListActive(ctx)
}

func (s *CouponService) Claim(ctx context.Context, userID, couponID string) (*models.Coupon, error) {
	coupon, err := s.repo.FindByID(ctx, couponID)
	if err != nil {
		return nil, err
	}
	if coupon.ValidUntil != nil && coupon.ValidUntil.Before(time.Now().UTC()) {
		return nil, ErrCouponExpired
	}
	if err := s.repo.Claim(ctx, userID, couponID); err != nil {
		return nil, err
	}
	return coupon, nil
}

func (s *CouponService) MyCoupons(ctx context.Context, userID string) ([]*models.UserCoupon, error) {
	return s.repo.ListClaimed(ctx, userID)
}

// Delete is administrator-only.
func (s *CouponService) Delete(ctx context.Context, identity Identity, id string) error {
	if identity.Role != roles.RoleAdministrator {
		return ErrForbidden
	}
	return s.repo.Delete(ctx, id)
}
