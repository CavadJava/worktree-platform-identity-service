package service

import (
	"context"
	"regexp"
	"strings"

	"shop-category-service/internal/models"
	"shop-category-service/internal/repository"
)

var (
	ErrCategoryNotFound    = repository.ErrCategoryNotFound
	ErrSubcategoryNotFound = repository.ErrSubcategoryNotFound
	ErrIDExists            = repository.ErrIDExists
)

var slugCleaner = regexp.MustCompile(`[^a-z0-9]+`)

// slugify turns "Women's Fashion" into "women-s-fashion" — used when the
// administrator creates an entry without an explicit id.
func slugify(s string) string {
	slug := slugCleaner.ReplaceAllString(strings.ToLower(strings.TrimSpace(s)), "-")
	return strings.Trim(slug, "-")
}

type CategoryService struct {
	repo *repository.CategoryRepository
}

func NewCategoryService(repo *repository.CategoryRepository) *CategoryService {
	return &CategoryService{repo: repo}
}

func (s *CategoryService) CreateCategory(ctx context.Context, id, name, icon string, sortOrder int) (*models.Category, error) {
	if id == "" {
		id = slugify(name)
	}
	c := &models.Category{ID: id, Name: strings.TrimSpace(name), Icon: icon, SortOrder: sortOrder}
	if err := s.repo.CreateCategory(ctx, c); err != nil {
		return nil, err
	}
	return s.repo.FindCategory(ctx, id)
}

func (s *CategoryService) ListCategories(ctx context.Context) ([]*models.Category, error) {
	return s.repo.ListCategories(ctx)
}

func (s *CategoryService) GetCategory(ctx context.Context, id string) (*models.Category, error) {
	return s.repo.FindCategory(ctx, id)
}

func (s *CategoryService) UpdateCategory(ctx context.Context, id, name, icon string, sortOrder int) (*models.Category, error) {
	return s.repo.UpdateCategory(ctx, id, strings.TrimSpace(name), icon, sortOrder)
}

func (s *CategoryService) DeleteCategory(ctx context.Context, id string) error {
	return s.repo.DeleteCategory(ctx, id)
}

func (s *CategoryService) CreateSubcategory(ctx context.Context, categoryID, id, name, image string, sortOrder int) (*models.Subcategory, error) {
	if _, err := s.repo.FindCategory(ctx, categoryID); err != nil {
		return nil, err
	}
	if id == "" {
		id = slugify(name)
	}
	sub := &models.Subcategory{ID: id, CategoryID: categoryID, Name: strings.TrimSpace(name), Image: image, SortOrder: sortOrder}
	if err := s.repo.CreateSubcategory(ctx, sub); err != nil {
		return nil, err
	}
	return s.repo.FindSubcategory(ctx, id)
}

func (s *CategoryService) ListSubcategories(ctx context.Context, categoryID string) ([]*models.Subcategory, error) {
	if categoryID != "" {
		if _, err := s.repo.FindCategory(ctx, categoryID); err != nil {
			return nil, err
		}
	}
	return s.repo.ListSubcategories(ctx, categoryID)
}

func (s *CategoryService) UpdateSubcategory(ctx context.Context, id, name, image string, sortOrder int) (*models.Subcategory, error) {
	return s.repo.UpdateSubcategory(ctx, id, strings.TrimSpace(name), image, sortOrder)
}

func (s *CategoryService) DeleteSubcategory(ctx context.Context, id string) error {
	return s.repo.DeleteSubcategory(ctx, id)
}
