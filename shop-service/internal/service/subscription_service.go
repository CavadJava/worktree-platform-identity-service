package service

import (
	"context"

	"shop-service/internal/models"
	"shop-service/internal/repository"
	"shop-service/internal/roles"
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

// ListSubscribers is the shop side of the relationship: any of the shop's
// own staff (chat(1)+, so the whole team — not owner-only) or a system
// administrator can see who's subscribed.
func (s *SubscriptionService) ListSubscribers(ctx context.Context, identity Identity, shopID string) ([]*models.Subscriber, error) {
	if !canManage(identity, shopID, roles.ShopLevelChat) {
		return nil, ErrForbidden
	}
	return s.subscriptions.ListByShop(ctx, shopID)
}
