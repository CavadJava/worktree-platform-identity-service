package service

import (
	"context"
	"errors"

	"platform-identity-service/internal/models"
	"platform-identity-service/internal/repository"
	"platform-identity-service/internal/service/shopassign"
)

var (
	ErrForbidden         = errors.New("forbidden")
	ErrInvalidSystemRole = errors.New("invalid system role: must be 'superadmin', 'admin', or 'user'")
	ErrInvalidStatus     = errors.New("invalid status: must be 'ACTIVE' or 'IN_ACTIVE'")
)

var systemRoleNameToID = map[string]int16{
	models.SystemRoleSuperadmin: 1,
	models.SystemRoleAdmin:      2,
	models.SystemRoleUser:       3,
}

type UserService struct {
	repo *repository.UserRepository
}

func NewUserService(repo *repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

// Get returns userID's record if caller is that same user or a superadmin.
// Shop-scoped viewing (e.g. a shop-admin seeing their own shop's members)
// goes through ShopMembershipService instead — this method is strictly
// about the system-wide user record.
func (s *UserService) Get(ctx context.Context, caller shopassign.Caller, userID string) (*models.User, error) {
	if caller.UserID != userID && caller.SystemRole != models.SystemRoleSuperadmin {
		return nil, ErrForbidden
	}
	return s.repo.GetByID(ctx, userID)
}

func (s *UserService) ListAll(ctx context.Context, caller shopassign.Caller) ([]models.User, error) {
	if caller.SystemRole != models.SystemRoleSuperadmin {
		return nil, ErrForbidden
	}
	return s.repo.ListAll(ctx)
}

func (s *UserService) SetSystemRole(ctx context.Context, caller shopassign.Caller, targetUserID, newSystemRoleName string) (*models.User, error) {
	if caller.SystemRole != models.SystemRoleSuperadmin {
		return nil, ErrForbidden
	}
	roleID, ok := systemRoleNameToID[newSystemRoleName]
	if !ok {
		return nil, ErrInvalidSystemRole
	}
	if err := s.repo.SetSystemRole(ctx, targetUserID, roleID); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, targetUserID)
}

func (s *UserService) SetStatus(ctx context.Context, caller shopassign.Caller, targetUserID, newStatus string) (*models.User, error) {
	if caller.SystemRole != models.SystemRoleSuperadmin {
		return nil, ErrForbidden
	}
	if newStatus != models.UserStatusActive && newStatus != models.UserStatusInActive {
		return nil, ErrInvalidStatus
	}
	if err := s.repo.SetStatus(ctx, targetUserID, newStatus); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, targetUserID)
}
