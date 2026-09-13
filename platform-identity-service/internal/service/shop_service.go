package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"platform-identity-service/internal/models"
	"platform-identity-service/internal/repository"
	"platform-identity-service/internal/service/shopassign"
)

var (
	ErrShopNotFound     = repository.ErrShopNotFound
	ErrInvalidShopType  = errors.New("invalid shop type: must be 'foreign' or 'local'")
)

type ShopService struct {
	repo *repository.ShopRepository
}

func NewShopService(repo *repository.ShopRepository) *ShopService {
	return &ShopService{repo: repo}
}

func (s *ShopService) Create(ctx context.Context, name, shopType string) (*models.Shop, error) {
	if shopType != models.ShopTypeForeign && shopType != models.ShopTypeLocal {
		return nil, ErrInvalidShopType
	}
	shop := &models.Shop{
		ID:        uuid.NewString(),
		Name:      name,
		ShopType:  shopType,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.repo.Create(ctx, shop); err != nil {
		return nil, err
	}
	return shop, nil
}

func (s *ShopService) List(ctx context.Context) ([]models.Shop, error) {
	return s.repo.List(ctx)
}

func (s *ShopService) Get(ctx context.Context, id string) (*models.Shop, error) {
	return s.repo.GetByID(ctx, id)
}

// UpdateProfile lets a superadmin set a shop's contact details (email,
// phone, address, work hours). Shop-scoped members (shop-admin/shop-user)
// don't manage this — it mirrors the product-profile pattern, where only
// the platform's own admin edits this kind of metadata.
func (s *ShopService) UpdateProfile(ctx context.Context, caller shopassign.Caller, shopID string, in repository.ShopProfileUpdate) (*models.Shop, error) {
	if caller.SystemRole != models.SystemRoleSuperadmin {
		return nil, ErrForbidden
	}
	if err := s.repo.UpdateProfile(ctx, shopID, in); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, shopID)
}
