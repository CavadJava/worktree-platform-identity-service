package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"platform-identity-service/internal/models"
	"platform-identity-service/internal/repository"
)

var ErrProjectNotFound = repository.ErrProjectNotFound

type ProjectService struct {
	repo *repository.ProjectRepository
}

func NewProjectService(repo *repository.ProjectRepository) *ProjectService {
	return &ProjectService{repo: repo}
}

func (s *ProjectService) Create(ctx context.Context, name string) (*models.Project, error) {
	p := &models.Project{
		ID:        uuid.NewString(),
		Name:      name,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *ProjectService) List(ctx context.Context) ([]models.Project, error) {
	return s.repo.List(ctx)
}

func (s *ProjectService) Get(ctx context.Context, id string) (*models.Project, error) {
	return s.repo.GetByID(ctx, id)
}
