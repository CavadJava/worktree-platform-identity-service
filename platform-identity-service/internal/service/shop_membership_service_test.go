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
	userSvc := NewUserService(userRepo)
	membershipSvc := NewShopMembershipService(membershipRepo, userRepo, shopRepo, authSvc, userSvc)
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

func TestShopMembershipService_SetMemberRole_ShopAdminCannotDemoteSelf(t *testing.T) {
	membershipSvc, authSvc, shopSvc := newTestShopMembershipService(t)

	shop, err := shopSvc.Create(context.Background(), "SelfDemote Test "+uuid.NewString())
	if err != nil {
		t.Fatalf("create shop failed: %v", err)
	}
	shopAdmin, err := authSvc.Register(context.Background(), RegisterInput{
		Name: "ShopAdmin", Username: "selfdemote-admin-" + uuid.NewString(), Email: uuid.NewString() + "@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("register shopAdmin failed: %v", err)
	}

	superadminCaller := shopassign.Caller{UserID: "superadmin-id", SystemRole: models.SystemRoleSuperadmin}
	if _, err := membershipSvc.AddMember(context.Background(), superadminCaller, shop.ID, shopAdmin.ID, models.ShopRoleAdmin); err != nil {
		t.Fatalf("bootstrap AddMember failed: %v", err)
	}

	shopAdminCaller := shopassign.Caller{UserID: shopAdmin.ID, SystemRole: models.SystemRoleUser}
	_, err = membershipSvc.SetMemberRole(context.Background(), shopAdminCaller, shop.ID, shopAdmin.ID, models.ShopRoleUser)
	if err != ErrCannotDemoteSelf {
		t.Errorf("expected ErrCannotDemoteSelf, got %v", err)
	}

	// A superadmin CAN still change that same shop-admin's role — the
	// self-lock only applies to the shop-admin acting on themself.
	updated, err := membershipSvc.SetMemberRole(context.Background(), superadminCaller, shop.ID, shopAdmin.ID, models.ShopRoleUser)
	if err != nil {
		t.Fatalf("superadmin SetMemberRole on same user failed: %v", err)
	}
	if updated.ShopRoleName != models.ShopRoleUser {
		t.Errorf("expected shop-user after superadmin change, got %q", updated.ShopRoleName)
	}
}

func TestShopMembershipService_RemoveMember(t *testing.T) {
	membershipSvc, authSvc, shopSvc := newTestShopMembershipService(t)

	shop, err := shopSvc.Create(context.Background(), "RemoveMember Test "+uuid.NewString())
	if err != nil {
		t.Fatalf("create shop failed: %v", err)
	}
	target, err := authSvc.Register(context.Background(), RegisterInput{
		Name: "Target", Username: "remove-target-" + uuid.NewString(), Email: uuid.NewString() + "@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("register target failed: %v", err)
	}

	superadminCaller := shopassign.Caller{UserID: "superadmin-id", SystemRole: models.SystemRoleSuperadmin}
	if _, err := membershipSvc.AddMember(context.Background(), superadminCaller, shop.ID, target.ID, models.ShopRoleUser); err != nil {
		t.Fatalf("AddMember failed: %v", err)
	}

	if err := membershipSvc.RemoveMember(context.Background(), superadminCaller, shop.ID, target.ID); err != nil {
		t.Fatalf("RemoveMember failed: %v", err)
	}

	members, err := membershipSvc.ListMembers(context.Background(), superadminCaller, shop.ID)
	if err != nil {
		t.Fatalf("ListMembers after remove failed: %v", err)
	}
	if len(members) != 0 {
		t.Errorf("expected 0 members after removal, got %d", len(members))
	}
}

func TestShopMembershipService_RemoveMember_ShopAdminCannotRemoveSelf(t *testing.T) {
	membershipSvc, authSvc, shopSvc := newTestShopMembershipService(t)

	shop, err := shopSvc.Create(context.Background(), "SelfRemove Test "+uuid.NewString())
	if err != nil {
		t.Fatalf("create shop failed: %v", err)
	}
	shopAdmin, err := authSvc.Register(context.Background(), RegisterInput{
		Name: "ShopAdmin", Username: "selfremove-admin-" + uuid.NewString(), Email: uuid.NewString() + "@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("register shopAdmin failed: %v", err)
	}

	superadminCaller := shopassign.Caller{UserID: "superadmin-id", SystemRole: models.SystemRoleSuperadmin}
	if _, err := membershipSvc.AddMember(context.Background(), superadminCaller, shop.ID, shopAdmin.ID, models.ShopRoleAdmin); err != nil {
		t.Fatalf("bootstrap AddMember failed: %v", err)
	}

	shopAdminCaller := shopassign.Caller{UserID: shopAdmin.ID, SystemRole: models.SystemRoleUser}
	if err := membershipSvc.RemoveMember(context.Background(), shopAdminCaller, shop.ID, shopAdmin.ID); err != ErrCannotDemoteSelf {
		t.Errorf("expected ErrCannotDemoteSelf, got %v", err)
	}
}

func TestShopMembershipService_UpdateMemberProfile(t *testing.T) {
	membershipSvc, authSvc, shopSvc := newTestShopMembershipService(t)

	shop, err := shopSvc.Create(context.Background(), "UpdateProfile Test "+uuid.NewString())
	if err != nil {
		t.Fatalf("create shop failed: %v", err)
	}
	shopAdmin, err := authSvc.Register(context.Background(), RegisterInput{
		Name: "ShopAdmin", Username: "profile-admin-" + uuid.NewString(), Email: uuid.NewString() + "@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("register shopAdmin failed: %v", err)
	}
	target, err := authSvc.Register(context.Background(), RegisterInput{
		Name: "Target", Username: "profile-target-" + uuid.NewString(), Email: uuid.NewString() + "@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("register target failed: %v", err)
	}

	superadminCaller := shopassign.Caller{UserID: "superadmin-id", SystemRole: models.SystemRoleSuperadmin}
	if _, err := membershipSvc.AddMember(context.Background(), superadminCaller, shop.ID, shopAdmin.ID, models.ShopRoleAdmin); err != nil {
		t.Fatalf("bootstrap shopAdmin failed: %v", err)
	}
	if _, err := membershipSvc.AddMember(context.Background(), superadminCaller, shop.ID, target.ID, models.ShopRoleUser); err != nil {
		t.Fatalf("AddMember target failed: %v", err)
	}

	newName := "Renamed Target"
	shopAdminCaller := shopassign.Caller{UserID: shopAdmin.ID, SystemRole: models.SystemRoleUser}
	updated, err := membershipSvc.UpdateMemberProfile(context.Background(), shopAdminCaller, shop.ID, target.ID, ProfileUpdate{Name: &newName})
	if err != nil {
		t.Fatalf("UpdateMemberProfile failed: %v", err)
	}
	if updated.Name != newName {
		t.Errorf("expected name %q, got %q", newName, updated.Name)
	}

	// A shop-admin of a DIFFERENT shop, or the target's own login as a
	// non-shop-admin, cannot edit — but the important case tested elsewhere
	// (AddMember_ForbiddenForNonMember) already covers the authorize()
	// path shared by this method, so it's not duplicated here.
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

func TestShopMembershipService_ListAllGroupedByUser(t *testing.T) {
	membershipSvc, authSvc, shopSvc := newTestShopMembershipService(t)

	shop, err := shopSvc.Create(context.Background(), "GroupedByUser Test "+uuid.NewString())
	if err != nil {
		t.Fatalf("create shop failed: %v", err)
	}
	withShop, err := authSvc.Register(context.Background(), RegisterInput{
		Name: "With Shop", Username: "grouped-with-" + uuid.NewString(), Email: uuid.NewString() + "@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("register withShop failed: %v", err)
	}
	withoutShop, err := authSvc.Register(context.Background(), RegisterInput{
		Name: "Without Shop", Username: "grouped-without-" + uuid.NewString(), Email: uuid.NewString() + "@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("register withoutShop failed: %v", err)
	}

	superadminCaller := shopassign.Caller{UserID: "superadmin-id", SystemRole: models.SystemRoleSuperadmin}
	if _, err := membershipSvc.AddMember(context.Background(), superadminCaller, shop.ID, withShop.ID, models.ShopRoleUser); err != nil {
		t.Fatalf("AddMember failed: %v", err)
	}

	grouped, err := membershipSvc.ListAllGroupedByUser(context.Background())
	if err != nil {
		t.Fatalf("ListAllGroupedByUser failed: %v", err)
	}
	if len(grouped[withShop.ID]) != 1 {
		t.Errorf("expected 1 membership for withShop, got %d", len(grouped[withShop.ID]))
	}
	if grouped[withShop.ID][0].ShopID != shop.ID {
		t.Errorf("expected membership's ShopID %q, got %q", shop.ID, grouped[withShop.ID][0].ShopID)
	}
	if _, ok := grouped[withoutShop.ID]; ok {
		t.Errorf("expected no entry for withoutShop (user with no shop memberships), got %v", grouped[withoutShop.ID])
	}
}
