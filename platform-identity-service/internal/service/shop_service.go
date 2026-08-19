package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"platform-identity-service/internal/models"
	"platform-identity-service/internal/repository"
)

var ErrShopNotFound = repository.ErrShopNotFound

type ShopService struct {
	repo *repository.ShopRepository
}

func NewShopService(repo *repository.ShopRepository) *ShopService {
	return &ShopService{repo: repo}
}

func (s *ShopService) Create(ctx context.Context, name string) (*models.Shop, error) {
	shop := &models.Shop{
		ID:        uuid.NewString(),
		Name:      name,
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
