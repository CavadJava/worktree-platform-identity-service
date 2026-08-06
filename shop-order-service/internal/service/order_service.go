package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"shop-order-service/internal/client"
	"shop-order-service/internal/models"
	"shop-order-service/internal/repository"
	"shop-order-service/internal/roles"
)

var (
	ErrItemsRequired          = errors.New("at least one item is required")
	ErrInvalidQuantity        = errors.New("quantity must be greater than zero")
	ErrProductShopMismatch    = errors.New("product item does not belong to the specified shop")
	ErrOrderNotFound          = repository.ErrOrderNotFound
	ErrProductItemNotFound    = repository.ErrProductItemNotFound
	ErrShopNotFound           = repository.ErrShopNotFound
	ErrUserNotFound           = repository.ErrUserNotFound
	ErrForbidden              = errors.New("forbidden")
	ErrInvalidStatus          = errors.New("unrecognized status")
	ErrStatusNotForward       = errors.New("status must move forward in the pipeline (pending → processing → shipped → in_transit → delivered), never backward or to the current stage")
	ErrCustomerCanOnlyConfirm = errors.New("a customer may only confirm delivery (set status to delivered), not any other stage")
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
	ProductItemID string
	Quantity      int
}

type OrderService struct {
	orders   *repository.OrderRepository
	lookup   *repository.LookupRepository
	payments *client.PaymentClient
}

func NewOrderService(orders *repository.OrderRepository, lookup *repository.LookupRepository, payments *client.PaymentClient) *OrderService {
	return &OrderService{orders: orders, lookup: lookup, payments: payments}
}

// Create validates every line item's product_item belongs to shopID,
// snapshots each item's current display name/price — resolving the
// discounted price server-side when the item is on sale, so later price or
// discount changes never rewrite past orders — and stamps an order_number
// of the form "U<user_seq>-S<shop_seq>-<n>" where <n> is this user's order
// count across all shops, plus one.
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
		productItem, err := s.lookup.GetProductItem(ctx, it.ProductItemID)
		if err != nil {
			return nil, err
		}
		if productItem.ShopID != shopID {
			return nil, ErrProductShopMismatch
		}
		total += productItem.Price * float64(it.Quantity)
		orderItems = append(orderItems, models.OrderItem{
			ID:            uuid.NewString(),
			OrderID:       orderID,
			ProductID:     productItem.ProductID,
			ProductItemID: productItem.ID,
			ProductName:   productItem.ProductName,
			ItemName:      productItem.ItemName,
			UnitPrice:     productItem.Price,
			Quantity:      it.Quantity,
			CreatedAt:     now,
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

	// Record the payment synchronously — the shop's balance should reflect
	// the sale the instant the order is placed, not after an eventual-
	// consistency lag. Still non-fatal: a payment-service hiccup must not
	// undo an order that's already been written (same "secondary concern
	// never breaks the primary write" rule notification fan-out follows
	// elsewhere in this codebase).
	if s.payments != nil {
		paymentItems := make([]client.PaymentItemInput, 0, len(orderItems))
		for _, it := range orderItems {
			paymentItems = append(paymentItems, client.PaymentItemInput{
				ProductID:   it.ProductID,
				ProductName: it.ProductName,
				Quantity:    it.Quantity,
				UnitPrice:   it.UnitPrice,
			})
		}
		if err := s.payments.RecordPayment(ctx, order.ID, userID, shopID, total, paymentItems); err != nil {
			log.Printf("record payment: order %s: %v", order.ID, err)
		}
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

// UpdateStatus moves an order forward through its delivery pipeline.
//
//   - The shop's own add-product(2)+ staff (or the system administrator)
//     can set it to any later stage, including delivered — this is "mağaza
//     təhvil verildi statusuna keçirir".
//   - The order's own customer can only ever set it to delivered — this is
//     "müştəri özü təsdiq edir" (confirming receipt), nothing else.
//   - Nobody can move it backward or re-set the current stage.
func (s *OrderService) UpdateStatus(ctx context.Context, caller Identity, orderID, newStatus string) (*models.Order, error) {
	order, err := s.orders.FindByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	newIdx := models.StatusIndex(newStatus)
	if newIdx == -1 {
		return nil, ErrInvalidStatus
	}

	isShopStaff := caller.isSystemAdmin() || caller.isShopStaffOf(order.ShopID)
	if !isShopStaff {
		if caller.UserID != order.UserID {
			return nil, ErrForbidden
		}
		if newStatus != models.StatusDelivered {
			return nil, ErrCustomerCanOnlyConfirm
		}
	}

	if newIdx <= models.StatusIndex(order.Status) {
		return nil, ErrStatusNotForward
	}

	return s.orders.UpdateStatus(ctx, orderID, newStatus)
}

func (s *OrderService) canView(caller Identity, order *models.Order) bool {
	if caller.UserID == order.UserID {
		return true
	}
	return caller.isSystemAdmin() || caller.isShopStaffOf(order.ShopID)
}
