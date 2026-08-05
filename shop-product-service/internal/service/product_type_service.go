package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"shop-product-service/internal/models"
	"shop-product-service/internal/repository"
	"shop-product-service/internal/roles"
)

var (
	ErrProductTypeNotFound    = repository.ErrProductTypeNotFound
	ErrProductSubtypeNotFound = repository.ErrProductSubtypeNotFound
)

type ProductTypeService struct {
	types    *repository.ProductTypeRepository
	subtypes *repository.ProductSubtypeRepository
}

func NewProductTypeService(types *repository.ProductTypeRepository, subtypes *repository.ProductSubtypeRepository) *ProductTypeService {
	return &ProductTypeService{types: types, subtypes: subtypes}
}

func (s *ProductTypeService) CreateType(ctx context.Context, identity Identity, shopID, name string) (*models.ProductType, error) {
	if !canManageShop(identity, shopID, roles.ShopLevelAddProduct) {
		return nil, ErrForbidden
	}
	now := time.Now().UTC()
	t := &models.ProductType{
		ID:        uuid.NewString(),
		ShopID:    shopID,
		Name:      strings.TrimSpace(name),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.types.Create(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *ProductTypeService) GetType(ctx context.Context, id string) (*models.ProductType, error) {
	return s.types.FindByID(ctx, id)
}

func (s *ProductTypeService) ListTypes(ctx context.Context, shopID string) ([]*models.ProductType, error) {
	return s.types.ListByShop(ctx, shopID)
}

func (s *ProductTypeService) UpdateType(ctx context.Context, identity Identity, id, name string) (*models.ProductType, error) {
	t, err := s.types.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !canManageShop(identity, t.ShopID, roles.ShopLevelAddProduct) {
		return nil, ErrForbidden
	}
	return s.types.Update(ctx, id, strings.TrimSpace(name))
}

func (s *ProductTypeService) DeleteType(ctx context.Context, identity Identity, id string) error {
	t, err := s.types.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if !canManageShop(identity, t.ShopID, roles.ShopLevelReview) {
		return ErrForbidden
	}
	return s.types.Delete(ctx, id)
}

func (s *ProductTypeService) CreateSubtype(ctx context.Context, identity Identity, productTypeID, name string) (*models.ProductSubtype, error) {
	t, err := s.types.FindByID(ctx, productTypeID)
	if err != nil {
		return nil, err
	}
	if !canManageShop(identity, t.ShopID, roles.ShopLevelAddProduct) {
		return nil, ErrForbidden
	}
	now := time.Now().UTC()
	st := &models.ProductSubtype{
		ID:            uuid.NewString(),
		ProductTypeID: productTypeID,
		Name:          strings.TrimSpace(name),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := s.subtypes.Create(ctx, st); err != nil {
		return nil, err
	}
	return st, nil
}

func (s *ProductTypeService) ListSubtypes(ctx context.Context, productTypeID string) ([]*models.ProductSubtype, error) {
	return s.subtypes.ListByType(ctx, productTypeID)
}

func (s *ProductTypeService) UpdateSubtype(ctx context.Context, identity Identity, id, name string) (*models.ProductSubtype, error) {
	st, err := s.subtypes.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	t, err := s.types.FindByID(ctx, st.ProductTypeID)
	if err != nil {
		return nil, err
	}
	if !canManageShop(identity, t.ShopID, roles.ShopLevelAddProduct) {
		return nil, ErrForbidden
	}
	return s.subtypes.Update(ctx, id, strings.TrimSpace(name))
}

func (s *ProductTypeService) DeleteSubtype(ctx context.Context, identity Identity, id string) error {
	st, err := s.subtypes.FindByID(ctx, id)
	if err != nil {
		return err
	}
	t, err := s.types.FindByID(ctx, st.ProductTypeID)
	if err != nil {
		return err
	}
	if !canManageShop(identity, t.ShopID, roles.ShopLevelReview) {
		return ErrForbidden
	}
	return s.subtypes.Delete(ctx, id)
}
