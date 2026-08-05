package service

import (
	"context"
	"errors"

	"shop-role-service/internal/repository"
	"shop-role-service/internal/roles"
)

var (
	ErrUserNotFound     = repository.ErrUserNotFound
	ErrInvalidShopLevel = errors.New("shop_role_level must be between 1 (chat) and 4 (admin)")
	ErrShopIDRequired   = errors.New("shop_id is required")
	ErrForbidden        = errors.New("forbidden")
)

type Assignment struct {
	UserID        string
	ShopID        *string
	ShopRoleLevel int
}

// Caller is whoever is invoking assign/revoke/get — either the system
// administrator (unrestricted) or a shop's own admin(4), who may only
// manage staff within that same shop.
type Caller struct {
	Role          string
	ShopID        *string
	ShopRoleLevel int
}

func (c Caller) isSystemAdmin() bool {
	return c.Role == roles.RoleAdministrator
}

func (c Caller) isShopOwnerOf(shopID string) bool {
	return c.ShopRoleLevel == roles.ShopLevelAdmin && c.ShopID != nil && *c.ShopID == shopID
}

type RoleService struct {
	repo *repository.RoleRepository
}

func NewRoleService(repo *repository.RoleRepository) *RoleService {
	return &RoleService{repo: repo}
}

func (s *RoleService) Get(ctx context.Context, caller Caller, userID string) (*Assignment, error) {
	shopID, level, err := s.repo.GetAssignment(ctx, userID)
	if err != nil {
		return nil, err
	}

	if !caller.isSystemAdmin() && !(shopID != nil && caller.isShopOwnerOf(*shopID)) {
		return nil, ErrForbidden
	}

	return &Assignment{UserID: userID, ShopID: shopID, ShopRoleLevel: level}, nil
}

// Assign sets a user's shop and hierarchical level in one shop: admin(4) >
// review(3) > add-product(2) > chat(1). A user belongs to at most one shop.
//
// The system administrator can assign anyone to any shop. A shop's own
// admin(4) may only assign staff into their own shop, and only users who
// aren't already committed to a different shop.
func (s *RoleService) Assign(ctx context.Context, caller Caller, userID, shopID string, level int) (*Assignment, error) {
	if shopID == "" {
		return nil, ErrShopIDRequired
	}
	if !roles.IsValidShopLevel(level) {
		return nil, ErrInvalidShopLevel
	}

	if !caller.isSystemAdmin() {
		if !caller.isShopOwnerOf(shopID) {
			return nil, ErrForbidden
		}
		currentShopID, _, err := s.repo.GetAssignment(ctx, userID)
		if err != nil {
			return nil, err
		}
		if currentShopID != nil && *currentShopID != shopID {
			return nil, ErrForbidden
		}
	}

	if err := s.repo.SetAssignment(ctx, userID, &shopID, level); err != nil {
		return nil, err
	}
	return &Assignment{UserID: userID, ShopID: &shopID, ShopRoleLevel: level}, nil
}

func (s *RoleService) Revoke(ctx context.Context, caller Caller, userID string) (*Assignment, error) {
	if !caller.isSystemAdmin() {
		currentShopID, _, err := s.repo.GetAssignment(ctx, userID)
		if err != nil {
			return nil, err
		}
		if currentShopID == nil || !caller.isShopOwnerOf(*currentShopID) {
			return nil, ErrForbidden
		}
	}

	if err := s.repo.SetAssignment(ctx, userID, nil, roles.ShopLevelNone); err != nil {
		return nil, err
	}
	return &Assignment{UserID: userID, ShopID: nil, ShopRoleLevel: roles.ShopLevelNone}, nil
}
