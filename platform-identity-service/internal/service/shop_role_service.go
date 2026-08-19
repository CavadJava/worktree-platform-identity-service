package service

import (
	"context"

	"platform-identity-service/internal/models"
	"platform-identity-service/internal/repository"
)

type ShopRoleService struct {
	repo *repository.ShopRoleRepository
}

func NewShopRoleService(repo *repository.ShopRoleRepository) *ShopRoleService {
	return &ShopRoleService{repo: repo}
}

func (s *ShopRoleService) List(ctx context.Context) ([]models.ShopRole, error) {
	return s.repo.List(ctx)
}
