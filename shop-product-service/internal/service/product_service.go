package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"shop-product-service/internal/models"
	"shop-product-service/internal/repository"
	"shop-product-service/internal/roles"
)

var (
	ErrProductNotFound     = repository.ErrProductNotFound
	ErrForbidden           = errors.New("forbidden")
	ErrProductTypeMismatch = errors.New("product type does not belong to this shop")
)

type Identity struct {
	UserID        string
	Role          string
	ShopID        *string
	ShopRoleLevel int
}

type ProductService struct {
	repo     *repository.ProductRepository
	typeRepo *repository.ProductTypeRepository
}

func NewProductService(repo *repository.ProductRepository, typeRepo *repository.ProductTypeRepository) *ProductService {
	return &ProductService{repo: repo, typeRepo: typeRepo}
}

func (s *ProductService) validateProductType(ctx context.Context, shopID string, productTypeID *string) error {
	if productTypeID == nil {
		return nil
	}
	pt, err := s.typeRepo.FindByID(ctx, *productTypeID)
	if err != nil {
		return err
	}
	if pt.ShopID != shopID {
		return ErrProductTypeMismatch
	}
	return nil
}

func (s *ProductService) Create(ctx context.Context, identity Identity, shopID, name, description string, price float64, stock int, productTypeID *string) (*models.Product, error) {
	if !canManageShop(identity, shopID, roles.ShopLevelAddProduct) {
		return nil, ErrForbidden
	}
	if err := s.validateProductType(ctx, shopID, productTypeID); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	p := &models.Product{
		ID:            uuid.NewString(),
		ShopID:        shopID,
		Name:          strings.TrimSpace(name),
		Description:   description,
		Price:         price,
		Stock:         stock,
		ProductTypeID: productTypeID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *ProductService) Get(ctx context.Context, id string) (*models.Product, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *ProductService) List(ctx context.Context, shopID string) ([]*models.Product, error) {
	return s.repo.List(ctx, shopID)
}

func (s *ProductService) Update(ctx context.Context, identity Identity, id, name, description string, price float64, stock int, productTypeID *string) (*models.Product, error) {
	product, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !canManageShop(identity, product.ShopID, roles.ShopLevelAddProduct) {
		return nil, ErrForbidden
	}
	if err := s.validateProductType(ctx, product.ShopID, productTypeID); err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, id, strings.TrimSpace(name), description, price, stock, productTypeID)
}

func (s *ProductService) Delete(ctx context.Context, identity Identity, id string) error {
	product, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if !canManageShop(identity, product.ShopID, roles.ShopLevelReview) {
		return ErrForbidden
	}
	return s.repo.Delete(ctx, id)
}

// canManageShop: administrators bypass entirely. Everyone else must belong
// to this exact shop (shop_id embedded in their JWT) with at least the
// required hierarchical level — no cross-service call needed, since
// shop_id/shop_role_level are only ever set by shop-service/shop-role-service
// on legitimate assignment.
func canManageShop(identity Identity, shopID string, requiredLevel int) bool {
	if identity.Role == roles.RoleAdministrator {
		return true
	}
	return identity.ShopID != nil && *identity.ShopID == shopID && identity.ShopRoleLevel >= requiredLevel
}
