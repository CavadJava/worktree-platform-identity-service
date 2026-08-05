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
	ErrProductItemNotFound  = repository.ErrProductItemNotFound
	ErrDiscountPriceMissing = errors.New("discount_price is required when is_discounted is true")
	ErrDiscountPriceInvalid = errors.New("discount_price must be lower than price")
)

type ProductItemService struct {
	items    *repository.ProductItemRepository
	products *repository.ProductRepository
}

func NewProductItemService(items *repository.ProductItemRepository, products *repository.ProductRepository) *ProductItemService {
	return &ProductItemService{items: items, products: products}
}

func (s *ProductItemService) Create(ctx context.Context, identity Identity, productID, name string, price float64, stock int, isDiscounted bool, discountPrice *float64) (*models.ProductItem, error) {
	product, err := s.products.FindByID(ctx, productID)
	if err != nil {
		return nil, err
	}
	if !canManageShop(identity, product.ShopID, roles.ShopLevelAddProduct) {
		return nil, ErrForbidden
	}
	if err := validateDiscount(isDiscounted, price, discountPrice); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	item := &models.ProductItem{
		ID:            uuid.NewString(),
		ProductID:     productID,
		Name:          strings.TrimSpace(name),
		Price:         price,
		Stock:         stock,
		IsDiscounted:  isDiscounted,
		DiscountPrice: discountPrice,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := s.items.Create(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *ProductItemService) Get(ctx context.Context, id string) (*models.ProductItem, error) {
	return s.items.FindByID(ctx, id)
}

func (s *ProductItemService) ListByProduct(ctx context.Context, productID string) ([]*models.ProductItem, error) {
	return s.items.ListByProduct(ctx, productID)
}

func (s *ProductItemService) Update(ctx context.Context, identity Identity, id, name string, price float64, stock int, isDiscounted bool, discountPrice *float64) (*models.ProductItem, error) {
	item, err := s.items.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	product, err := s.products.FindByID(ctx, item.ProductID)
	if err != nil {
		return nil, err
	}
	if !canManageShop(identity, product.ShopID, roles.ShopLevelAddProduct) {
		return nil, ErrForbidden
	}
	if err := validateDiscount(isDiscounted, price, discountPrice); err != nil {
		return nil, err
	}

	return s.items.Update(ctx, id, strings.TrimSpace(name), price, stock, isDiscounted, discountPrice)
}

func (s *ProductItemService) Delete(ctx context.Context, identity Identity, id string) error {
	item, err := s.items.FindByID(ctx, id)
	if err != nil {
		return err
	}
	product, err := s.products.FindByID(ctx, item.ProductID)
	if err != nil {
		return err
	}
	if !canManageShop(identity, product.ShopID, roles.ShopLevelReview) {
		return ErrForbidden
	}
	return s.items.Delete(ctx, id)
}

func validateDiscount(isDiscounted bool, price float64, discountPrice *float64) error {
	if !isDiscounted {
		return nil
	}
	if discountPrice == nil {
		return ErrDiscountPriceMissing
	}
	if *discountPrice >= price {
		return ErrDiscountPriceInvalid
	}
	return nil
}
