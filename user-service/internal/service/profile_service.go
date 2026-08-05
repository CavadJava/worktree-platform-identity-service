package service

import (
	"context"

	"user-service/internal/models"
	"user-service/internal/repository"
)

var ErrUserNotFound = repository.ErrUserNotFound

type ProfileService struct {
	repo *repository.UserRepository
}

func NewProfileService(repo *repository.UserRepository) *ProfileService {
	return &ProfileService{repo: repo}
}

func (s *ProfileService) GetProfile(ctx context.Context, userID string) (*models.User, error) {
	return s.repo.FindByID(ctx, userID)
}

func (s *ProfileService) UpdateProfile(ctx context.Context, userID, fullName, phone string) (*models.User, error) {
	return s.repo.UpdateProfile(ctx, userID, fullName, phone)
}
