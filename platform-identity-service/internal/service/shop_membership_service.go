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
	ErrCannotDemoteSelf   = errors.New("a shop-admin cannot change their own shop role")
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
	userSvc        *UserService
}

func NewShopMembershipService(membershipRepo *repository.ShopMembershipRepository, userRepo *repository.UserRepository, shopRepo *repository.ShopRepository, authSvc *AuthService, userSvc *UserService) *ShopMembershipService {
	return &ShopMembershipService{membershipRepo: membershipRepo, userRepo: userRepo, shopRepo: shopRepo, authSvc: authSvc, userSvc: userSvc}
}

// callerMembership fetches caller's own membership row for shopID, or nil
// if they have none — shared by authorize and authorizeView below.
// Superadmins skip the lookup entirely — both shopassign checks always
// allow them regardless of membership, and a superadmin caller's UserID
// is not guaranteed to be a real user_shop_memberships row (or even a
// valid UUID).
func (s *ShopMembershipService) callerMembership(ctx context.Context, caller shopassign.Caller, shopID string) (*models.ShopMembership, error) {
	if caller.SystemRole == models.SystemRoleSuperadmin {
		return nil, nil
	}
	m, err := s.membershipRepo.GetByUserAndShop(ctx, caller.UserID, shopID)
	if errors.Is(err, repository.ErrMembershipNotFound) {
		return nil, nil
	}
	return m, err
}

// authorize gates management actions (add/remove/change-role) — only a
// superadmin or that shop's own shop-admin passes.
func (s *ShopMembershipService) authorize(ctx context.Context, caller shopassign.Caller, shopID string) error {
	m, err := s.callerMembership(ctx, caller, shopID)
	if err != nil {
		return err
	}
	if !shopassign.CanManageShop(caller, m) {
		return ErrForbidden
	}
	return nil
}

// authorizeView gates read-only access (listing members) — a plain
// shop-user of that shop passes too, not just shop-admin/superadmin.
func (s *ShopMembershipService) authorizeView(ctx context.Context, caller shopassign.Caller, shopID string) error {
	m, err := s.callerMembership(ctx, caller, shopID)
	if err != nil {
		return err
	}
	if !shopassign.CanViewShop(caller, m) {
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
	if err := s.authorizeView(ctx, caller, shopID); err != nil {
		return nil, err
	}
	return s.membershipRepo.ListByShop(ctx, shopID)
}

func (s *ShopMembershipService) SetMemberRole(ctx context.Context, caller shopassign.Caller, shopID, targetUserID, newShopRoleName string) (*models.ShopMembership, error) {
	if err := s.authorize(ctx, caller, shopID); err != nil {
		return nil, err
	}
	// A superadmin's UserID may be a synthetic placeholder, not a real
	// user id — the self-check only makes sense for a real shop-admin
	// caller, so skip it for superadmin the same way authorize() does.
	if caller.SystemRole != models.SystemRoleSuperadmin && caller.UserID == targetUserID {
		return nil, ErrCannotDemoteSelf
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

// RemoveMember removes targetUserID's membership from shopID — the user's
// Teslahubs account itself is untouched, they simply stop belonging to
// this shop. Same authority as SetMemberRole (shop-admin of this shop, or
// superadmin), with the same self-removal guard.
func (s *ShopMembershipService) RemoveMember(ctx context.Context, caller shopassign.Caller, shopID, targetUserID string) error {
	if err := s.authorize(ctx, caller, shopID); err != nil {
		return err
	}
	if caller.SystemRole != models.SystemRoleSuperadmin && caller.UserID == targetUserID {
		return ErrCannotDemoteSelf
	}
	return s.membershipRepo.Delete(ctx, targetUserID, shopID)
}

// UpdateMemberProfile lets a shop's own shop-admin (or superadmin) edit a
// member's name/email/password. Uses the same authorization as
// AddMember/SetMemberRole (shop-admin of THIS shop, or superadmin) —
// targetUserID must actually be a member of shopID, otherwise this would
// let a shop-admin edit an arbitrary user by guessing their id.
func (s *ShopMembershipService) UpdateMemberProfile(ctx context.Context, caller shopassign.Caller, shopID, targetUserID string, in ProfileUpdate) (*models.User, error) {
	if err := s.authorize(ctx, caller, shopID); err != nil {
		return nil, err
	}
	if _, err := s.membershipRepo.GetByUserAndShop(ctx, targetUserID, shopID); err != nil {
		return nil, err
	}
	return s.userSvc.applyProfileUpdate(ctx, targetUserID, in)
}

// ListMyShops returns every shop userID belongs to — no authorization
// check beyond "you can always see your own memberships," since the
// handler layer restricts this to the caller viewing their own id or a
// superadmin viewing anyone's.
func (s *ShopMembershipService) ListMyShops(ctx context.Context, userID string) ([]models.ShopMembership, error) {
	return s.membershipRepo.ListByUser(ctx, userID)
}

// ListAllGroupedByUser returns every membership across every shop, keyed
// by user id — used to enrich the system-wide user list (UserHandler.ListAll)
// with each user's shop memberships in one extra query, instead of an
// N+1 lookup per user. Caller (superadmin-only, enforced by UserService.ListAll
// before this is ever reached) is not re-checked here.
func (s *ShopMembershipService) ListAllGroupedByUser(ctx context.Context) (map[string][]models.ShopMembership, error) {
	all, err := s.membershipRepo.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	grouped := make(map[string][]models.ShopMembership)
	for _, m := range all {
		grouped[m.UserID] = append(grouped[m.UserID], m)
	}
	return grouped, nil
}
