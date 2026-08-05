package service

import (
	"context"

	"shop-product-service/internal/models"
	"shop-product-service/internal/repository"
)

type FavoriteService struct {
	favorites *repository.FavoriteRepository
	products  *repository.ProductRepository
}

func NewFavoriteService(favorites *repository.FavoriteRepository, products *repository.ProductRepository) *FavoriteService {
	return &FavoriteService{favorites: favorites, products: products}
}

// Add favorites a product for a user. The product must exist — this is the
// only check, favoriting has no ownership/level restriction (any logged-in
// user can favorite any product).
func (s *FavoriteService) Add(ctx context.Context, userID, productID string) error {
	if _, err := s.products.FindByID(ctx, productID); err != nil {
		return err
	}
	return s.favorites.Add(ctx, userID, productID)
}

func (s *FavoriteService) Remove(ctx context.Context, userID, productID string) error {
	return s.favorites.Remove(ctx, userID, productID)
}

func (s *FavoriteService) List(ctx context.Context, userID string) ([]*models.Product, error) {
	return s.favorites.ListByUser(ctx, userID)
}
