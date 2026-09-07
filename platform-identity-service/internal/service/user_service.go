package service

import (
	"context"
	"errors"

	"platform-identity-service/internal/auth"
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

// ListBasic is a narrower listing than ListAll: no shop-membership data,
// available to admin as well as superadmin — an admin managing a product
// needs to pick a subject user (for a subscription or an admin-request)
// without gaining visibility into everyone's shop memberships, which
// ListAll's superadmin-only gate exists specifically to protect.
func (s *UserService) ListBasic(ctx context.Context, caller shopassign.Caller) ([]models.User, error) {
	if caller.SystemRole != models.SystemRoleAdmin && caller.SystemRole != models.SystemRoleSuperadmin {
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

// ProfileUpdate mirrors repository.UserUpdate at the service boundary —
// nil fields are left unchanged, and Password (plaintext, if provided) is
// hashed here before ever reaching the repository.
type ProfileUpdate struct {
	Name     *string
	Email    *string
	Password *string
}

// UpdateProfile lets a user edit their own name/email/password, or lets a
// superadmin edit anyone's. Editing a shop-mate's profile as a shop-admin
// goes through ShopMembershipService.UpdateMemberProfile instead, since
// that authorization decision needs the shop-membership lookup this
// service doesn't have.
func (s *UserService) UpdateProfile(ctx context.Context, caller shopassign.Caller, targetUserID string, in ProfileUpdate) (*models.User, error) {
	if caller.UserID != targetUserID && caller.SystemRole != models.SystemRoleSuperadmin {
		return nil, ErrForbidden
	}
	return s.applyProfileUpdate(ctx, targetUserID, in)
}

// applyProfileUpdate does the actual hash-and-write, shared by
// UserService.UpdateProfile and ShopMembershipService.UpdateMemberProfile
// (which perform their own, different authorization checks before calling
// this — UserService exports it as a package-level helper via this method
// rather than duplicating the hashing logic).
func (s *UserService) applyProfileUpdate(ctx context.Context, targetUserID string, in ProfileUpdate) (*models.User, error) {
	update := repository.UserUpdate{Name: in.Name, Email: in.Email}
	if in.Password != nil {
		hash, err := auth.HashPassword(*in.Password)
		if err != nil {
			return nil, err
		}
		update.PasswordHash = &hash
	}
	if err := s.repo.Update(ctx, targetUserID, update); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, targetUserID)
}
