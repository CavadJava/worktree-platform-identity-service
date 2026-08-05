package service

import (
	"context"

	"shop-service/internal/models"
	"shop-service/internal/repository"
)

type SubscriptionService struct {
	subscriptions *repository.SubscriptionRepository
	shops         *repository.ShopRepository
}

func NewSubscriptionService(subscriptions *repository.SubscriptionRepository, shops *repository.ShopRepository) *SubscriptionService {
	return &SubscriptionService{subscriptions: subscriptions, shops: shops}
}

// Subscribe follows a shop. The shop must exist — any logged-in user can
// subscribe to any shop, no ownership/level restriction.
func (s *SubscriptionService) Subscribe(ctx context.Context, userID, shopID string) error {
	if _, err := s.shops.FindByID(ctx, shopID); err != nil {
		return err
	}
	return s.subscriptions.Add(ctx, userID, shopID)
}

func (s *SubscriptionService) Unsubscribe(ctx context.Context, userID, shopID string) error {
	return s.subscriptions.Remove(ctx, userID, shopID)
}

func (s *SubscriptionService) List(ctx context.Context, userID string) ([]*models.Shop, error) {
	return s.subscriptions.ListByUser(ctx, userID)
}
