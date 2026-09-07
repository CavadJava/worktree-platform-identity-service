package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"platform-identity-service/internal/models"
	"platform-identity-service/internal/repository"
	"platform-identity-service/internal/service/shopassign"
)

var ErrProductNotFound = repository.ErrProductNotFound

const (
	AccessFull = "full"
	AccessDemo = "demo"
)

type ProductService struct {
	productRepo *repository.ProductRepository
	subRepo     *repository.SubscriptionRepository
	subprojRepo *repository.SubprojectRepository
	authSvc     *AuthService
}

func NewProductService(productRepo *repository.ProductRepository, subRepo *repository.SubscriptionRepository, subprojRepo *repository.SubprojectRepository, authSvc *AuthService) *ProductService {
	return &ProductService{productRepo: productRepo, subRepo: subRepo, subprojRepo: subprojRepo, authSvc: authSvc}
}

// canManage reports whether caller may administer productID: a superadmin
// always may; an admin may only for a product they hold a subscription to.
// A plain user is never allowed, and never reaches this far in practice
// since the HTTP layer requires an authenticated caller with a system role
// of admin or superadmin to reach any of these handlers at all.
func (s *ProductService) canManage(ctx context.Context, caller shopassign.Caller, productID string) error {
	if caller.SystemRole == models.SystemRoleSuperadmin {
		return nil
	}
	if caller.SystemRole != models.SystemRoleAdmin {
		return ErrForbidden
	}
	_, err := s.subRepo.GetByUserAndProduct(ctx, caller.UserID, productID)
	if errors.Is(err, repository.ErrSubscriptionNotFound) {
		return ErrForbidden
	}
	if err != nil {
		return err
	}
	return nil
}

func (s *ProductService) Create(ctx context.Context, name string) (*models.Product, error) {
	p := &models.Product{ID: uuid.NewString(), Name: name, CreatedAt: time.Now().UTC()}
	if err := s.productRepo.Create(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *ProductService) List(ctx context.Context) ([]models.Product, error) {
	return s.productRepo.List(ctx)
}

// UpdateProfile lets a superadmin or an admin who holds a subscription to the product set a product's description and tech
// stack. Both fields are always submitted together by the admin panel form.
func (s *ProductService) UpdateProfile(ctx context.Context, caller shopassign.Caller, productID, description, techStack string) (*models.Product, error) {
	if err := s.canManage(ctx, caller, productID); err != nil {
		return nil, err
	}
	if err := s.productRepo.Update(ctx, productID, description, techStack); err != nil {
		return nil, err
	}
	return s.productRepo.GetByID(ctx, productID)
}

// AddSubproject lets a superadmin or an admin who holds a subscription to the product record a named internal module of a
// product (e.g. Teslahubs -> auth-service). Manually entered, no external
// discovery.
func (s *ProductService) AddSubproject(ctx context.Context, caller shopassign.Caller, productID, name, description string) (*models.ProductSubproject, error) {
	if err := s.canManage(ctx, caller, productID); err != nil {
		return nil, err
	}
	if _, err := s.productRepo.GetByID(ctx, productID); err != nil {
		return nil, err
	}
	sub := &models.ProductSubproject{
		ID: uuid.NewString(), ProductID: productID, Name: name, Description: description,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.subprojRepo.Create(ctx, sub); err != nil {
		return nil, err
	}
	return sub, nil
}

// ListSubprojects has no caller-role check, matching List's existing
// no-auth read pattern — subprojects are visible to anyone who can see the
// product.
func (s *ProductService) ListSubprojects(ctx context.Context, productID string) ([]models.ProductSubproject, error) {
	return s.subprojRepo.ListByProduct(ctx, productID)
}

func (s *ProductService) RemoveSubproject(ctx context.Context, caller shopassign.Caller, subprojectID string) error {
	sub, err := s.subprojRepo.GetByID(ctx, subprojectID)
	if err != nil {
		return err
	}
	if err := s.canManage(ctx, caller, sub.ProductID); err != nil {
		return err
	}
	return s.subprojRepo.Delete(ctx, subprojectID)
}

// CheckAccess reports "full" if userID has an active subscription to
// productID, "demo" otherwise (including when no subscription row exists
// at all — a user who never subscribed still gets demo access, not an
// error).
func (s *ProductService) CheckAccess(ctx context.Context, userID, productID string) (string, error) {
	if _, err := s.productRepo.GetByID(ctx, productID); err != nil {
		return "", err
	}

	sub, err := s.subRepo.GetByUserAndProduct(ctx, userID, productID)
	if errors.Is(err, repository.ErrSubscriptionNotFound) {
		return AccessDemo, nil
	}
	if err != nil {
		return "", err
	}
	if sub.Subscripted {
		return AccessFull, nil
	}
	return AccessDemo, nil
}

// SetSubscription is called by an admin/superadmin to manually toggle a
// user's subscription — there is no payment gateway integration in this
// scope.
func (s *ProductService) SetSubscription(ctx context.Context, caller shopassign.Caller, targetUserID, productID string, subscripted, renewed bool, notes string) (*models.Subscription, error) {
	if err := s.canManage(ctx, caller, productID); err != nil {
		return nil, err
	}
	if _, err := s.productRepo.GetByID(ctx, productID); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	existing, err := s.subRepo.GetByUserAndProduct(ctx, targetUserID, productID)
	id := uuid.NewString()
	createdAt := now
	if err == nil {
		id = existing.ID
		createdAt = existing.CreatedAt
	} else if !errors.Is(err, repository.ErrSubscriptionNotFound) {
		return nil, err
	}

	sub := &models.Subscription{
		ID: id, UserID: targetUserID, ProductID: productID,
		Subscripted: subscripted, Renewed: renewed, Notes: notes,
		CreatedAt: createdAt, UpdatedAt: now,
	}
	if err := s.subRepo.Upsert(ctx, sub); err != nil {
		return nil, err
	}
	return sub, nil
}

// CreateUserAndSubscribe creates a brand-new Teslahubs account (system role
// 'user', same as self-service Register) and immediately subscribes it to
// productID — a superadmin or admin who holds a subscription onboarding someone straight into a product from
// the admin panel, without a separate "create user, then find them in a
// list, then subscribe them" round trip.
func (s *ProductService) CreateUserAndSubscribe(ctx context.Context, caller shopassign.Caller, productID string, in CreateUserInput) (*models.Subscription, error) {
	if err := s.canManage(ctx, caller, productID); err != nil {
		return nil, err
	}
	if _, err := s.productRepo.GetByID(ctx, productID); err != nil {
		return nil, err
	}

	in.SystemRole = models.SystemRoleUser
	u, err := s.authSvc.CreateUser(ctx, in)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	sub := &models.Subscription{
		ID: uuid.NewString(), UserID: u.ID, ProductID: productID,
		Subscripted: true, Renewed: false,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := s.subRepo.Upsert(ctx, sub); err != nil {
		return nil, err
	}
	return sub, nil
}
