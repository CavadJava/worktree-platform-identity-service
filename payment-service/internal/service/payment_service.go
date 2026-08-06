package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"payment-service/internal/models"
	"payment-service/internal/repository"
	"payment-service/internal/roles"
)

var (
	ErrItemsRequired        = errors.New("at least one item is required")
	ErrInvalidAmount        = errors.New("amount must be greater than zero")
	ErrPaymentAlreadyExists = repository.ErrPaymentAlreadyExists
	ErrForbidden            = errors.New("forbidden")
)

// Identity is the calling user's claims, as returned by authorization-service.
type Identity struct {
	UserID        string
	Role          string
	ShopID        *string
	ShopRoleLevel int
}

// canView: administrators bypass entirely; a shop's own chat(1)+ staff (the
// whole team, not owner-only — same reasoning as review-service's and
// shop-service's subscriber list) can see their own shop's payments/balance.
func canView(identity Identity, shopID string) bool {
	if identity.Role == roles.RoleAdministrator {
		return true
	}
	return identity.ShopID != nil && *identity.ShopID == shopID && identity.ShopRoleLevel >= roles.ShopLevelChat
}

type ItemInput struct {
	ProductID   string
	ProductName string
	Quantity    int
	UnitPrice   float64
}

type PaymentService struct {
	repo *repository.PaymentRepository
}

func NewPaymentService(repo *repository.PaymentRepository) *PaymentService {
	return &PaymentService{repo: repo}
}

// Create records a completed payment and credits the shop's balance —
// called once, server-to-server, by shop-order-service right after it
// creates an order (see that service's PaymentClient). Unauthenticated by
// design, same convention as notification-service's POST /notifications:
// this is an internal call, not something an end user's token would sign.
func (s *PaymentService) Create(ctx context.Context, orderID, userID, shopID string, amount float64, items []ItemInput) (*models.Payment, error) {
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}
	if len(items) == 0 {
		return nil, ErrItemsRequired
	}

	now := time.Now().UTC()
	paymentID := uuid.NewString()
	paymentItems := make([]models.PaymentItem, 0, len(items))
	for _, it := range items {
		paymentItems = append(paymentItems, models.PaymentItem{
			ID:          uuid.NewString(),
			PaymentID:   paymentID,
			ProductID:   it.ProductID,
			ProductName: it.ProductName,
			Quantity:    it.Quantity,
			UnitPrice:   it.UnitPrice,
		})
	}

	payment := &models.Payment{
		ID:        paymentID,
		OrderID:   orderID,
		UserID:    userID,
		ShopID:    shopID,
		Amount:    amount,
		Status:    models.StatusCompleted,
		Items:     paymentItems,
		CreatedAt: now,
	}

	if err := s.repo.Create(ctx, payment); err != nil {
		return nil, err
	}
	return payment, nil
}

func (s *PaymentService) ListByShop(ctx context.Context, identity Identity, shopID string) ([]*models.Payment, error) {
	if !canView(identity, shopID) {
		return nil, ErrForbidden
	}
	return s.repo.ListByShop(ctx, shopID)
}

func (s *PaymentService) GetBalance(ctx context.Context, identity Identity, shopID string) (*models.ShopBalance, error) {
	if !canView(identity, shopID) {
		return nil, ErrForbidden
	}
	return s.repo.GetBalance(ctx, shopID)
}
