package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"platform-identity-service/internal/models"
	"platform-identity-service/internal/repository"
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
