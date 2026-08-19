package service

import (
	"context"

	"platform-identity-service/internal/models"
	"platform-identity-service/internal/repository"
)

type SystemRoleService struct {
	repo *repository.SystemRoleRepository
}

func NewSystemRoleService(repo *repository.SystemRoleRepository) *SystemRoleService {
	return &SystemRoleService{repo: repo}
}

func (s *SystemRoleService) List(ctx context.Context) ([]models.SystemRole, error) {
	return s.repo.List(ctx)
}
