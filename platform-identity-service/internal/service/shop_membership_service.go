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

var (
	ErrMembershipNotFound = repository.ErrMembershipNotFound
	ErrMembershipExists   = repository.ErrMembershipExists
	ErrInvalidShopRole    = errors.New("invalid shop role: must be 'shop-admin' or 'shop-user'")
)

var shopRoleNameToID = map[string]int16{
	models.ShopRoleAdmin: 1,
	models.ShopRoleUser:  2,
}

type ShopMembershipService struct {
	membershipRepo *repository.ShopMembershipRepository
	userRepo       *repository.UserRepository
	shopRepo       *repository.ShopRepository
	authSvc        *AuthService
}

func NewShopMembershipService(membershipRepo *repository.ShopMembershipRepository, userRepo *repository.UserRepository, shopRepo *repository.ShopRepository, authSvc *AuthService) *ShopMembershipService {
	return &ShopMembershipService{membershipRepo: membershipRepo, userRepo: userRepo, shopRepo: shopRepo, authSvc: authSvc}
}

// authorize fetches caller's own membership row for shopID (nil if none)
// and evaluates shopassign.CanManageShop against it. Superadmins skip the
// membership lookup entirely — CanManageShop always allows them regardless
// of membership, and a superadmin caller's UserID is not guaranteed to be
// a real user_shop_memberships row (or even a valid UUID).
func (s *ShopMembershipService) authorize(ctx context.Context, caller shopassign.Caller, shopID string) error {
	if caller.SystemRole == models.SystemRoleSuperadmin {
		return nil
	}

	callerMembership, err := s.membershipRepo.GetByUserAndShop(ctx, caller.UserID, shopID)
	if err != nil && !errors.Is(err, repository.ErrMembershipNotFound) {
		return err
	}
	if errors.Is(err, repository.ErrMembershipNotFound) {
		callerMembership = nil
	}
	if !shopassign.CanManageShop(caller, callerMembership) {
		return ErrForbidden
	}
	return nil
}

func (s *ShopMembershipService) AddMember(ctx context.Context, caller shopassign.Caller, shopID, targetUserID, shopRoleName string) (*models.ShopMembership, error) {
	if err := s.authorize(ctx, caller, shopID); err != nil {
		return nil, err
	}

	shopRoleID, ok := shopRoleNameToID[shopRoleName]
	if !ok {
		return nil, ErrInvalidShopRole
	}

	if _, err := s.shopRepo.GetByID(ctx, shopID); err != nil {
		return nil, err
	}
	if _, err := s.userRepo.GetByID(ctx, targetUserID); err != nil {
		return nil, err
	}

	m := &models.ShopMembership{
		ID:         uuid.NewString(),
		UserID:     targetUserID,
		ShopID:     shopID,
		ShopRoleID: shopRoleID,
		CreatedAt:  time.Now().UTC(),
	}
	if err := s.membershipRepo.Create(ctx, m); err != nil {
		return nil, err
	}
	return s.membershipRepo.GetByUserAndShop(ctx, targetUserID, shopID)
}

// AddNewMember creates a brand-new Teslahubs account (system role 'user',
// same as self-service Register) and immediately adds it to shopID as
// shop-user — a shop's own shop-admin uses this to onboard someone who
// doesn't have an account yet, without needing a separate registration
// step or knowing the person's user id in advance.
func (s *ShopMembershipService) AddNewMember(ctx context.Context, caller shopassign.Caller, shopID string, in CreateUserInput) (*models.ShopMembership, error) {
	if err := s.authorize(ctx, caller, shopID); err != nil {
		return nil, err
	}

	in.SystemRole = models.SystemRoleUser
	u, err := s.authSvc.CreateUser(ctx, in)
	if err != nil {
		return nil, err
	}

	m := &models.ShopMembership{
		ID:         uuid.NewString(),
		UserID:     u.ID,
		ShopID:     shopID,
		ShopRoleID: shopRoleNameToID[models.ShopRoleUser],
		CreatedAt:  time.Now().UTC(),
	}
	if err := s.membershipRepo.Create(ctx, m); err != nil {
		return nil, err
	}
	return s.membershipRepo.GetByUserAndShop(ctx, u.ID, shopID)
}

func (s *ShopMembershipService) ListMembers(ctx context.Context, caller shopassign.Caller, shopID string) ([]models.ShopMembership, error) {
	if err := s.authorize(ctx, caller, shopID); err != nil {
		return nil, err
	}
	return s.membershipRepo.ListByShop(ctx, shopID)
}

func (s *ShopMembershipService) SetMemberRole(ctx context.Context, caller shopassign.Caller, shopID, targetUserID, newShopRoleName string) (*models.ShopMembership, error) {
	if err := s.authorize(ctx, caller, shopID); err != nil {
		return nil, err
	}

	shopRoleID, ok := shopRoleNameToID[newShopRoleName]
	if !ok {
		return nil, ErrInvalidShopRole
	}

	if err := s.membershipRepo.SetShopRole(ctx, targetUserID, shopID, shopRoleID); err != nil {
		return nil, err
	}
	return s.membershipRepo.GetByUserAndShop(ctx, targetUserID, shopID)
}

// ListMyShops returns every shop userID belongs to — no authorization
// check beyond "you can always see your own memberships," since the
// handler layer restricts this to the caller viewing their own id or a
// superadmin viewing anyone's.
func (s *ShopMembershipService) ListMyShops(ctx context.Context, userID string) ([]models.ShopMembership, error) {
	return s.membershipRepo.ListByUser(ctx, userID)
}
