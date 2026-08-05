package service

import (
	"context"
	"errors"
	"fmt"
	"log"
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

// Notifier sends a notification to a single user — satisfied by
// client.NotificationClient. Kept as an interface so the service layer
// stays decoupled from the HTTP client.
type Notifier interface {
	Send(notificationType, userID, message string) error
}

type ProductService struct {
	repo        *repository.ProductRepository
	typeRepo    *repository.ProductTypeRepository
	subscribers *repository.SubscriberRepository
	notifier    Notifier
}

func NewProductService(repo *repository.ProductRepository, typeRepo *repository.ProductTypeRepository, subscribers *repository.SubscriberRepository, notifier Notifier) *ProductService {
	return &ProductService{repo: repo, typeRepo: typeRepo, subscribers: subscribers, notifier: notifier}
}

// notifySubscribers fans a "new product" notification out to everyone
// subscribed to the shop. Runs in its own goroutine with a fresh context
// (the request context is gone by then) and is strictly best-effort:
// failures are logged, never surfaced — a missed notification must not
// affect the product that was already created.
func (s *ProductService) notifySubscribers(shopID, productName string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userIDs, err := s.subscribers.ListUserIDsByShop(ctx, shopID)
	if err != nil {
		log.Printf("notify subscribers: list failed for shop %s: %v", shopID, err)
		return
	}
	if len(userIDs) == 0 {
		return
	}

	shopName, err := s.subscribers.ShopName(ctx, shopID)
	if err != nil {
		shopName = "Abunə olduğunuz mağaza"
	}
	message := fmt.Sprintf("%s yeni məhsul əlavə etdi: %s", shopName, productName)

	for _, userID := range userIDs {
		if err := s.notifier.Send("new_product", userID, message); err != nil {
			log.Printf("notify subscribers: send to %s failed: %v", userID, err)
		}
	}
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

	go s.notifySubscribers(p.ShopID, p.Name)

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
