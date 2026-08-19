package service

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"platform-identity-service/internal/models"
	"platform-identity-service/internal/repository"
	"platform-identity-service/internal/service/shopassign"
)

func newTestShopMembershipService(t *testing.T) (*ShopMembershipService, *AuthService, *ShopService) {
	t.Helper()
	db := testDB(t)
	userRepo := repository.NewUserRepository(db)
	shopRepo := repository.NewShopRepository(db)
	membershipRepo := repository.NewShopMembershipRepository(db)
	jwtMgr := newTestJWTManager()
	authSvc := NewAuthService(userRepo, jwtMgr)
	shopSvc := NewShopService(shopRepo)
	membershipSvc := NewShopMembershipService(membershipRepo, userRepo, shopRepo, authSvc)
	return membershipSvc, authSvc, shopSvc
}

func TestShopMembershipService_AddMember_BySuperadmin(t *testing.T) {
	membershipSvc, authSvc, shopSvc := newTestShopMembershipService(t)

	shop, err := shopSvc.Create(context.Background(), "Superadmin Add Test "+uuid.NewString())
	if err != nil {
		t.Fatalf("create shop failed: %v", err)
	}
	target, err := authSvc.Register(context.Background(), RegisterInput{
		Name: "Target", Username: "target-" + uuid.NewString(), Email: uuid.NewString() + "@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("register target failed: %v", err)
	}

	superadminCaller := shopassign.Caller{UserID: "superadmin-id", SystemRole: models.SystemRoleSuperadmin}
	membership, err := membershipSvc.AddMember(context.Background(), superadminCaller, shop.ID, target.ID, models.ShopRoleUser)
	if err != nil {
		t.Fatalf("AddMember failed: %v", err)
	}
	if membership.ShopRoleName != models.ShopRoleUser {
		t.Errorf("expected shop-user, got %q", membership.ShopRoleName)
	}
}

func TestShopMembershipService_AddMember_ByShopAdmin(t *testing.T) {
	membershipSvc, authSvc, shopSvc := newTestShopMembershipService(t)

	shop, err := shopSvc.Create(context.Background(), "ShopAdmin Add Test "+uuid.NewString())
	if err != nil {
		t.Fatalf("create shop failed: %v", err)
	}
	shopAdmin, err := authSvc.Register(context.Background(), RegisterInput{
		Name: "ShopAdmin", Username: "shopadmin-" + uuid.NewString(), Email: uuid.NewString() + "@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("register shopAdmin failed: %v", err)
	}
	// Bootstrap: make shopAdmin a shop-admin of this shop via a superadmin call.
	superadminCaller := shopassign.Caller{UserID: "superadmin-id", SystemRole: models.SystemRoleSuperadmin}
	if _, err := membershipSvc.AddMember(context.Background(), superadminCaller, shop.ID, shopAdmin.ID, models.ShopRoleAdmin); err != nil {
		t.Fatalf("bootstrap AddMember failed: %v", err)
	}

	target, err := authSvc.Register(context.Background(), RegisterInput{
		Name: "Target", Username: "target-" + uuid.NewString(), Email: uuid.NewString() + "@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("register target failed: %v", err)
	}

	shopAdminCaller := shopassign.Caller{UserID: shopAdmin.ID, SystemRole: models.SystemRoleUser}
	membership, err := membershipSvc.AddMember(context.Background(), shopAdminCaller, shop.ID, target.ID, models.ShopRoleUser)
	if err != nil {
		t.Fatalf("AddMember by shop-admin failed: %v", err)
	}
	if membership.ShopRoleName != models.ShopRoleUser {
		t.Errorf("expected shop-user, got %q", membership.ShopRoleName)
	}
}

func TestShopMembershipService_AddNewMember_ByShopAdmin(t *testing.T) {
	membershipSvc, authSvc, shopSvc := newTestShopMembershipService(t)

	shop, err := shopSvc.Create(context.Background(), "ShopAdmin AddNew Test "+uuid.NewString())
	if err != nil {
		t.Fatalf("create shop failed: %v", err)
	}
	shopAdmin, err := authSvc.Register(context.Background(), RegisterInput{
		Name: "ShopAdmin", Username: "shopadmin-new-" + uuid.NewString(), Email: uuid.NewString() + "@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("register shopAdmin failed: %v", err)
	}
	superadminCaller := shopassign.Caller{UserID: "superadmin-id", SystemRole: models.SystemRoleSuperadmin}
	if _, err := membershipSvc.AddMember(context.Background(), superadminCaller, shop.ID, shopAdmin.ID, models.ShopRoleAdmin); err != nil {
		t.Fatalf("bootstrap AddMember failed: %v", err)
	}

	shopAdminCaller := shopassign.Caller{UserID: shopAdmin.ID, SystemRole: models.SystemRoleUser}
	membership, err := membershipSvc.AddNewMember(context.Background(), shopAdminCaller, shop.ID, CreateUserInput{
		Name: "Fresh", Username: "fresh-" + uuid.NewString(), Email: uuid.NewString() + "@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("AddNewMember failed: %v", err)
	}
	if membership.ShopRoleName != models.ShopRoleUser {
		t.Errorf("expected shop-user, got %q", membership.ShopRoleName)
	}
}

func TestShopMembershipService_AddNewMember_ForbiddenForNonMember(t *testing.T) {
	membershipSvc, authSvc, shopSvc := newTestShopMembershipService(t)

	shop, err := shopSvc.Create(context.Background(), "AddNewMember Forbidden Test "+uuid.NewString())
	if err != nil {
		t.Fatalf("create shop failed: %v", err)
	}
	caller, err := authSvc.Register(context.Background(), RegisterInput{
		Name: "Nobody", Username: "nobody-new-" + uuid.NewString(), Email: uuid.NewString() + "@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("register caller failed: %v", err)
	}

	plainCaller := shopassign.Caller{UserID: caller.ID, SystemRole: models.SystemRoleUser}
	_, err = membershipSvc.AddNewMember(context.Background(), plainCaller, shop.ID, CreateUserInput{
		Name: "New", Username: "new-" + uuid.NewString(), Email: uuid.NewString() + "@example.com", Password: "password123",
	})
	if err != ErrForbidden {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestShopMembershipService_AddMember_ForbiddenForNonMember(t *testing.T) {
	membershipSvc, authSvc, shopSvc := newTestShopMembershipService(t)

	shop, err := shopSvc.Create(context.Background(), "Forbidden Add Test "+uuid.NewString())
	if err != nil {
		t.Fatalf("create shop failed: %v", err)
	}
	caller, err := authSvc.Register(context.Background(), RegisterInput{
		Name: "Nobody", Username: "nobody-" + uuid.NewString(), Email: uuid.NewString() + "@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("register caller failed: %v", err)
	}
	target, err := authSvc.Register(context.Background(), RegisterInput{
		Name: "Target", Username: "target-" + uuid.NewString(), Email: uuid.NewString() + "@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("register target failed: %v", err)
	}

	plainCaller := shopassign.Caller{UserID: caller.ID, SystemRole: models.SystemRoleUser}
	_, err = membershipSvc.AddMember(context.Background(), plainCaller, shop.ID, target.ID, models.ShopRoleUser)
	if err != ErrForbidden {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestShopMembershipService_ListMembers(t *testing.T) {
	membershipSvc, authSvc, shopSvc := newTestShopMembershipService(t)

	shop, err := shopSvc.Create(context.Background(), "ListMembers Test "+uuid.NewString())
	if err != nil {
		t.Fatalf("create shop failed: %v", err)
	}
	target, err := authSvc.Register(context.Background(), RegisterInput{
		Name: "Target", Username: "target-" + uuid.NewString(), Email: uuid.NewString() + "@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("register target failed: %v", err)
	}

	superadminCaller := shopassign.Caller{UserID: "superadmin-id", SystemRole: models.SystemRoleSuperadmin}
	if _, err := membershipSvc.AddMember(context.Background(), superadminCaller, shop.ID, target.ID, models.ShopRoleUser); err != nil {
		t.Fatalf("AddMember failed: %v", err)
	}

	members, err := membershipSvc.ListMembers(context.Background(), superadminCaller, shop.ID)
	if err != nil {
		t.Fatalf("ListMembers failed: %v", err)
	}
	if len(members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(members))
	}
}

func TestShopMembershipService_ListMembers_PlainShopUserCanView(t *testing.T) {
	membershipSvc, authSvc, shopSvc := newTestShopMembershipService(t)

	shop, err := shopSvc.Create(context.Background(), "ListMembers ShopUser Test "+uuid.NewString())
	if err != nil {
		t.Fatalf("create shop failed: %v", err)
	}
	member, err := authSvc.Register(context.Background(), RegisterInput{
		Name: "Member", Username: "member-view-" + uuid.NewString(), Email: uuid.NewString() + "@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("register member failed: %v", err)
	}

	superadminCaller := shopassign.Caller{UserID: "superadmin-id", SystemRole: models.SystemRoleSuperadmin}
	if _, err := membershipSvc.AddMember(context.Background(), superadminCaller, shop.ID, member.ID, models.ShopRoleUser); err != nil {
		t.Fatalf("AddMember failed: %v", err)
	}

	// A plain shop-user (not shop-admin) can view the member list, even
	// though they can't manage it (AddMember/SetMemberRole stay shop-admin-only).
	memberCaller := shopassign.Caller{UserID: member.ID, SystemRole: models.SystemRoleUser}
	members, err := membershipSvc.ListMembers(context.Background(), memberCaller, shop.ID)
	if err != nil {
		t.Fatalf("ListMembers as shop-user failed: %v", err)
	}
	if len(members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(members))
	}

	if _, err := membershipSvc.SetMemberRole(context.Background(), memberCaller, shop.ID, member.ID, models.ShopRoleAdmin); err != ErrForbidden {
		t.Errorf("expected shop-user to be forbidden from SetMemberRole, got %v", err)
	}
}

func TestShopMembershipService_SetMemberRole(t *testing.T) {
	membershipSvc, authSvc, shopSvc := newTestShopMembershipService(t)

	shop, err := shopSvc.Create(context.Background(), "SetMemberRole Test "+uuid.NewString())
	if err != nil {
		t.Fatalf("create shop failed: %v", err)
	}
	target, err := authSvc.Register(context.Background(), RegisterInput{
		Name: "Target", Username: "target-" + uuid.NewString(), Email: uuid.NewString() + "@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("register target failed: %v", err)
	}

	superadminCaller := shopassign.Caller{UserID: "superadmin-id", SystemRole: models.SystemRoleSuperadmin}
	if _, err := membershipSvc.AddMember(context.Background(), superadminCaller, shop.ID, target.ID, models.ShopRoleUser); err != nil {
		t.Fatalf("AddMember failed: %v", err)
	}

	updated, err := membershipSvc.SetMemberRole(context.Background(), superadminCaller, shop.ID, target.ID, models.ShopRoleAdmin)
	if err != nil {
		t.Fatalf("SetMemberRole failed: %v", err)
	}
	if updated.ShopRoleName != models.ShopRoleAdmin {
		t.Errorf("expected shop-admin, got %q", updated.ShopRoleName)
	}
}

func TestShopMembershipService_ListMyShops(t *testing.T) {
	membershipSvc, authSvc, shopSvc := newTestShopMembershipService(t)

	shopA, err := shopSvc.Create(context.Background(), "MyShops A "+uuid.NewString())
	if err != nil {
		t.Fatalf("create shopA failed: %v", err)
	}
	shopB, err := shopSvc.Create(context.Background(), "MyShops B "+uuid.NewString())
	if err != nil {
		t.Fatalf("create shopB failed: %v", err)
	}
	target, err := authSvc.Register(context.Background(), RegisterInput{
		Name: "Target", Username: "target-" + uuid.NewString(), Email: uuid.NewString() + "@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("register target failed: %v", err)
	}

	superadminCaller := shopassign.Caller{UserID: "superadmin-id", SystemRole: models.SystemRoleSuperadmin}
	if _, err := membershipSvc.AddMember(context.Background(), superadminCaller, shopA.ID, target.ID, models.ShopRoleAdmin); err != nil {
		t.Fatalf("AddMember shopA failed: %v", err)
	}
	if _, err := membershipSvc.AddMember(context.Background(), superadminCaller, shopB.ID, target.ID, models.ShopRoleUser); err != nil {
		t.Fatalf("AddMember shopB failed: %v", err)
	}

	myShops, err := membershipSvc.ListMyShops(context.Background(), target.ID)
	if err != nil {
		t.Fatalf("ListMyShops failed: %v", err)
	}
	if len(myShops) != 2 {
		t.Fatalf("expected 2 shop memberships, got %d", len(myShops))
	}
}
