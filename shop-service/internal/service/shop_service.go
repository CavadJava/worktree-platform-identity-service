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
	ErrShopNotFound = repository.ErrShopNotFound
	ErrForbidden    = errors.New("forbidden")
)

type Identity struct {
	UserID        string
	Role          string
	ShopID        *string
	ShopRoleLevel int
}

type ShopService struct {
	repo *repository.ShopRepository
}

func NewShopService(repo *repository.ShopRepository) *ShopService {
	return &ShopService{repo: repo}
}

// Create is administrator-only — regular users open a shop through the
// application/approval flow (see ShopApplicationService), not directly.
func (s *ShopService) Create(ctx context.Context, identity Identity, name, description string) (*models.Shop, error) {
	if identity.Role != roles.RoleAdministrator {
		return nil, ErrForbidden
	}
	return s.create(ctx, identity.UserID, name, description)
}

func (s *ShopService) create(ctx context.Context, ownerID, name, description string) (*models.Shop, error) {
	now := time.Now().UTC()
	shop := &models.Shop{
		ID:          uuid.NewString(),
		OwnerID:     ownerID,
		Name:        strings.TrimSpace(name),
		Description: description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.Create(ctx, shop); err != nil {
		return nil, err
	}
	return shop, nil
}

func (s *ShopService) Get(ctx context.Context, id string) (*models.Shop, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *ShopService) List(ctx context.Context, ownerID string) ([]*models.Shop, error) {
	return s.repo.List(ctx, ownerID)
}

func (s *ShopService) Update(ctx context.Context, identity Identity, id, name, description string) (*models.Shop, error) {
	if !canManage(identity, id, roles.ShopLevelReview) {
		return nil, ErrForbidden
	}
	return s.repo.Update(ctx, id, strings.TrimSpace(name), description)
}

func (s *ShopService) Delete(ctx context.Context, identity Identity, id string) error {
	if !canManage(identity, id, roles.ShopLevelAdmin) {
		return ErrForbidden
	}
	return s.repo.Delete(ctx, id)
}

// canManage: administrators bypass entirely; everyone else must belong to
// this exact shop with at least the required hierarchical level.
func canManage(identity Identity, shopID string, requiredLevel int) bool {
	if identity.Role == roles.RoleAdministrator {
		return true
	}
	return identity.ShopID != nil && *identity.ShopID == shopID && identity.ShopRoleLevel >= requiredLevel
}
