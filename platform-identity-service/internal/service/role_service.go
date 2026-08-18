package service

import (
	"context"

	"platform-identity-service/internal/models"
	"platform-identity-service/internal/repository"
)

type RoleService struct {
	repo *repository.RoleRepository
}

func NewRoleService(repo *repository.RoleRepository) *RoleService {
	return &RoleService{repo: repo}
}

func (s *RoleService) List(ctx context.Context) ([]models.Role, error) {
	return s.repo.List(ctx)
}
