package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"shop-order-service/internal/models"
	"shop-order-service/internal/repository"
	"shop-order-service/internal/roles"
)

var (
	ErrItemsRequired       = errors.New("at least one item is required")
	ErrInvalidQuantity     = errors.New("quantity must be greater than zero")
	ErrProductShopMismatch = errors.New("product does not belong to the specified shop")
	ErrOrderNotFound       = repository.ErrOrderNotFound
	ErrProductNotFound     = repository.ErrProductNotFound
	ErrShopNotFound        = repository.ErrShopNotFound
	ErrUserNotFound        = repository.ErrUserNotFound
	ErrForbidden           = errors.New("forbidden")
)

// Identity is the calling user's claims, as returned by authorization-service.
type Identity struct {
	UserID        string
	Role          string
	ShopID        *string
	ShopRoleLevel int
}

func (i Identity) isSystemAdmin() bool {
	return i.Role == roles.RoleAdministrator
}

func (i Identity) isShopStaffOf(shopID string) bool {
	return i.ShopRoleLevel >= roles.ShopLevelAddProduct && i.ShopID != nil && *i.ShopID == shopID
}

type ItemInput struct {
	ProductID string
	Quantity  int
}

type OrderService struct {
	orders *repository.OrderRepository
	lookup *repository.LookupRepository
}

func NewOrderService(orders *repository.OrderRepository, lookup *repository.LookupRepository) *OrderService {
	return &OrderService{orders: orders, lookup: lookup}
}

// Create validates every line item belongs to shopID, snapshots each
// product's current name/price (so later price changes don't rewrite past
// orders), and stamps an order_number of the form "U<user_seq>-S<shop_seq>-<n>"
// where <n> is this user's order count across all shops, plus one.
func (s *OrderService) Create(ctx context.Context, userID, shopID string, items []ItemInput) (*models.Order, error) {
	if len(items) == 0 {
		return nil, ErrItemsRequired
	}
	for _, it := range items {
		if it.Quantity <= 0 {
			return nil, ErrInvalidQuantity
		}
	}

	shopSeq, err := s.lookup.GetShopSeq(ctx, shopID)
	if err != nil {
		return nil, err
	}
	userSeq, err := s.lookup.GetUserSeq(ctx, userID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	orderID := uuid.NewString()
	orderItems := make([]models.OrderItem, 0, len(items))
	var total float64
	for _, it := range items {
		product, err := s.lookup.GetProduct(ctx, it.ProductID)
		if err != nil {
			return nil, err
		}
		if product.ShopID != shopID {
			return nil, ErrProductShopMismatch
		}
		total += product.Price * float64(it.Quantity)
		orderItems = append(orderItems, models.OrderItem{
			ID:          uuid.NewString(),
			OrderID:     orderID,
			ProductID:   product.ID,
			ProductName: product.Name,
			UnitPrice:   product.Price,
			Quantity:    it.Quantity,
			CreatedAt:   now,
		})
	}

	priorCount, err := s.orders.CountByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	order := &models.Order{
		ID:          orderID,
		OrderNumber: fmt.Sprintf("U%d-S%d-%d", userSeq, shopSeq, priorCount+1),
		UserID:      userID,
		ShopID:      shopID,
		Status:      models.StatusPending,
		TotalAmount: total,
		Items:       orderItems,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.orders.Create(ctx, order); err != nil {
		return nil, err
	}
	return order, nil
}

func (s *OrderService) Get(ctx context.Context, caller Identity, id string) (*models.Order, error) {
	order, err := s.orders.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !s.canView(caller, order) {
		return nil, ErrForbidden
	}
	return order, nil
}

func (s *OrderService) ListMine(ctx context.Context, userID string) ([]*models.Order, error) {
	return s.orders.ListByUser(ctx, userID)
}

// ListForShop returns a shop's incoming orders — the caller must be that
// shop's add-product(2)+ staff, or the system administrator.
func (s *OrderService) ListForShop(ctx context.Context, caller Identity, shopID string) ([]*models.Order, error) {
	if !caller.isSystemAdmin() && !caller.isShopStaffOf(shopID) {
		return nil, ErrForbidden
	}
	return s.orders.ListByShop(ctx, shopID)
}

func (s *OrderService) canView(caller Identity, order *models.Order) bool {
	if caller.UserID == order.UserID {
		return true
	}
	return caller.isSystemAdmin() || caller.isShopStaffOf(order.ShopID)
}
