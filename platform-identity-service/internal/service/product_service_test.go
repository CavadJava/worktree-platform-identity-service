package service

import (
	"context"
	"errors"
	"testing"

	"platform-identity-service/internal/models"
	"platform-identity-service/internal/repository"
	"platform-identity-service/internal/service/shopassign"
)

func newTestProductService(t *testing.T) *ProductService {
	t.Helper()
	db := testDB(t)
	userRepo := repository.NewUserRepository(db)
	productRepo := repository.NewProductRepository(db)
	subRepo := repository.NewSubscriptionRepository(db)
	subprojRepo := repository.NewSubprojectRepository(db)
	reqRepo := repository.NewProductAdminRequestRepository(db)
	authSvc := NewAuthService(userRepo, newTestJWTManager())
	return NewProductService(productRepo, subRepo, subprojRepo, reqRepo, userRepo, authSvc)
}

func TestProductService_UpdateProfile_RequiresSuperadmin(t *testing.T) {
	svc := newTestProductService(t)
	p, err := svc.Create(context.Background(), "Teslahubs")
	if err != nil {
		t.Fatalf("create product: %v", err)
	}

	_, err = svc.UpdateProfile(context.Background(), shopassign.Caller{SystemRole: models.SystemRoleUser}, p.ID, "desc", "React")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden for non-superadmin, got %v", err)
	}

	updated, err := svc.UpdateProfile(context.Background(), shopassign.Caller{SystemRole: models.SystemRoleSuperadmin}, p.ID, "A Tesla platform", "React, Go, Postgres")
	if err != nil {
		t.Fatalf("update profile: %v", err)
	}
	if updated.Description != "A Tesla platform" || updated.TechStack != "React, Go, Postgres" {
		t.Fatalf("expected updated profile, got %+v", updated)
	}
}

func TestProductService_CreateUserAndSubscribe(t *testing.T) {
	svc := newTestProductService(t)
	p, err := svc.Create(context.Background(), "Teslahubs")
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	superadmin := shopassign.Caller{SystemRole: models.SystemRoleSuperadmin}
	nonAdmin := shopassign.Caller{SystemRole: models.SystemRoleUser}

	in := CreateUserInput{Name: "New User", Username: "newuser-" + p.ID, Email: "newuser-" + p.ID + "@example.com", Password: "password123"}

	if _, err := svc.CreateUserAndSubscribe(context.Background(), nonAdmin, p.ID, in, models.SystemRoleUser); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden for non-superadmin, got %v", err)
	}

	sub, err := svc.CreateUserAndSubscribe(context.Background(), superadmin, p.ID, in, models.SystemRoleUser)
	if err != nil {
		t.Fatalf("create user and subscribe: %v", err)
	}
	if !sub.Subscripted {
		t.Fatalf("expected new user to be subscripted, got %+v", sub)
	}

	access, err := svc.CheckAccess(context.Background(), sub.UserID, p.ID)
	if err != nil {
		t.Fatalf("check access: %v", err)
	}
	if access != AccessFull {
		t.Fatalf("expected full access after subscribe, got %q", access)
	}
}

func TestProductService_SubprojectLifecycle(t *testing.T) {
	svc := newTestProductService(t)
	p, err := svc.Create(context.Background(), "Teslahubs")
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	superadmin := shopassign.Caller{SystemRole: models.SystemRoleSuperadmin}
	nonAdmin := shopassign.Caller{SystemRole: models.SystemRoleUser}

	if _, err := svc.AddSubproject(context.Background(), nonAdmin, p.ID, "auth-service", "handles login"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden for non-superadmin add, got %v", err)
	}

	sub, err := svc.AddSubproject(context.Background(), superadmin, p.ID, "auth-service", "handles login")
	if err != nil {
		t.Fatalf("add subproject: %v", err)
	}

	list, err := svc.ListSubprojects(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("list subprojects: %v", err)
	}
	if len(list) != 1 || list[0].Name != "auth-service" {
		t.Fatalf("expected one subproject, got %+v", list)
	}

	if err := svc.RemoveSubproject(context.Background(), nonAdmin, sub.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden for non-superadmin remove, got %v", err)
	}
	if err := svc.RemoveSubproject(context.Background(), superadmin, sub.ID); err != nil {
		t.Fatalf("remove subproject: %v", err)
	}

	list, err = svc.ListSubprojects(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("list subprojects after remove: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected zero subprojects after remove, got %d", len(list))
	}
}

func TestProductService_AdminWithSubscriptionCanManageTheirProduct(t *testing.T) {
	svc := newTestProductService(t)
	p, err := svc.Create(context.Background(), "Teslahubs")
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	superadmin := shopassign.Caller{SystemRole: models.SystemRoleSuperadmin}

	// Create an admin user
	adminUser, err := svc.authSvc.CreateUser(context.Background(), CreateUserInput{
		Name:       "Admin User",
		Username:   "admin-" + p.ID,
		Email:      "admin-" + p.ID + "@example.com",
		Password:   "password123",
		SystemRole: models.SystemRoleAdmin,
	})
	if err != nil {
		t.Fatalf("create admin user: %v", err)
	}
	admin := shopassign.Caller{UserID: adminUser.ID, SystemRole: models.SystemRoleAdmin}

	// Admin has no subscription yet — must be forbidden.
	if _, err := svc.UpdateProfile(context.Background(), admin, p.ID, "desc", "Go"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden before subscription exists, got %v", err)
	}

	// Superadmin subscribes the admin to the product.
	if _, err := svc.SetSubscription(context.Background(), superadmin, admin.UserID, p.ID, true, false, ""); err != nil {
		t.Fatalf("subscribe admin: %v", err)
	}

	// Now the admin can manage it.
	updated, err := svc.UpdateProfile(context.Background(), admin, p.ID, "A Tesla platform", "Go, React")
	if err != nil {
		t.Fatalf("expected admin with subscription to manage product, got %v", err)
	}
	if updated.Description != "A Tesla platform" {
		t.Fatalf("expected profile updated, got %+v", updated)
	}

	// A different admin, still unsubscribed, is still forbidden.
	otherAdminUser, err := svc.authSvc.CreateUser(context.Background(), CreateUserInput{
		Name:       "Other Admin",
		Username:   "other-admin-" + p.ID,
		Email:      "other-admin-" + p.ID + "@example.com",
		Password:   "password123",
		SystemRole: models.SystemRoleAdmin,
	})
	if err != nil {
		t.Fatalf("create other admin user: %v", err)
	}
	otherAdmin := shopassign.Caller{UserID: otherAdminUser.ID, SystemRole: models.SystemRoleAdmin}
	if _, err := svc.UpdateProfile(context.Background(), otherAdmin, p.ID, "x", "y"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden for unsubscribed admin, got %v", err)
	}
}

func TestProductService_ListCustomers(t *testing.T) {
	svc := newTestProductService(t)
	p, err := svc.Create(context.Background(), "Teslahubs")
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	other, err := svc.Create(context.Background(), "ESound")
	if err != nil {
		t.Fatalf("create other product: %v", err)
	}
	superadmin := shopassign.Caller{SystemRole: models.SystemRoleSuperadmin}

	adminUser, err := svc.authSvc.CreateUser(context.Background(), CreateUserInput{
		Name: "Owner Admin", Username: "owner-" + p.ID, Email: "owner-" + p.ID + "@example.com",
		Password: "password123", SystemRole: models.SystemRoleAdmin,
	})
	if err != nil {
		t.Fatalf("create admin user: %v", err)
	}
	admin := shopassign.Caller{UserID: adminUser.ID, SystemRole: models.SystemRoleAdmin}

	// Not subscribed yet — forbidden.
	if _, err := svc.ListCustomers(context.Background(), admin, p.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden before subscription exists, got %v", err)
	}

	if _, err := svc.SetSubscription(context.Background(), superadmin, admin.UserID, p.ID, true, false, ""); err != nil {
		t.Fatalf("subscribe admin: %v", err)
	}

	customerSub, err := svc.CreateUserAndSubscribe(context.Background(), superadmin, p.ID, CreateUserInput{
		Name: "Plain Customer", Username: "customer-" + p.ID, Email: "customer-" + p.ID + "@example.com", Password: "password123",
	}, models.SystemRoleUser)
	if err != nil {
		t.Fatalf("create customer: %v", err)
	}

	list, err := svc.ListCustomers(context.Background(), admin, p.ID)
	if err != nil {
		t.Fatalf("expected admin with subscription to list customers, got %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 customers (admin + plain customer), got %d", len(list))
	}
	foundCustomer := false
	for _, c := range list {
		if c.UserID == customerSub.UserID {
			foundCustomer = true
		}
	}
	if !foundCustomer {
		t.Fatalf("expected created customer in list, got %+v", list)
	}

	// Admin cannot list a product they don't manage.
	if _, err := svc.ListCustomers(context.Background(), admin, other.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden for unmanaged product, got %v", err)
	}

	// Superadmin can list any product's customers, no subscription needed.
	if _, err := svc.ListCustomers(context.Background(), superadmin, p.ID); err != nil {
		t.Fatalf("expected superadmin to list customers, got %v", err)
	}
}

func TestProductService_RequestApprovePromoteFlow(t *testing.T) {
	svc := newTestProductService(t)
	p, err := svc.Create(context.Background(), "Teslahubs")
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	superadminUser, err := svc.authSvc.CreateUser(context.Background(), CreateUserInput{
		Name:       "Superadmin",
		Username:   "superadmin-" + p.ID,
		Email:      "superadmin-" + p.ID + "@example.com",
		Password:   "password123",
		SystemRole: models.SystemRoleSuperadmin,
	})
	if err != nil {
		t.Fatalf("create superadmin user: %v", err)
	}
	superadmin := shopassign.Caller{UserID: superadminUser.ID, SystemRole: models.SystemRoleSuperadmin}

	// Create a plain 'user' subject via the existing product-scoped creation path.
	newUserSub, err := svc.CreateUserAndSubscribe(context.Background(), superadmin, p.ID, CreateUserInput{
		Name: "Subject", Username: "subject-" + p.ID, Email: "subject-" + p.ID + "@example.com", Password: "password123",
	}, models.SystemRoleUser)
	if err != nil {
		t.Fatalf("create subject user: %v", err)
	}
	_ = newUserSub

	// The subject already has a subscription (from CreateUserAndSubscribe), so
	// build a second, unsubscribed target product to test the request flow on.
	p2, err := svc.Create(context.Background(), "ESound")
	if err != nil {
		t.Fatalf("create second product: %v", err)
	}

	adminUser, err := svc.authSvc.CreateUser(context.Background(), CreateUserInput{
		Name:       "Requesting Admin",
		Username:   "req-admin-" + p.ID,
		Email:      "req-admin-" + p.ID + "@example.com",
		Password:   "password123",
		SystemRole: models.SystemRoleAdmin,
	})
	if err != nil {
		t.Fatalf("create admin user: %v", err)
	}
	admin := shopassign.Caller{UserID: adminUser.ID, SystemRole: models.SystemRoleAdmin}
	// admin has no subscription to p2 yet, so filing a request must fail.
	if _, err := svc.RequestProductAdmin(context.Background(), admin, p2.ID, "some-user-id"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden for admin unsubscribed to the product, got %v", err)
	}

	// Subscribe the admin to p2, then they can file a request for someone else.
	if _, err := svc.SetSubscription(context.Background(), superadmin, admin.UserID, p2.ID, true, false, ""); err != nil {
		t.Fatalf("subscribe admin to p2: %v", err)
	}

	req, err := svc.RequestProductAdmin(context.Background(), admin, p2.ID, newUserSub.UserID)
	if err != nil {
		t.Fatalf("request product admin: %v", err)
	}
	if req.Status != models.ProductAdminRequestPending {
		t.Fatalf("expected pending status, got %q", req.Status)
	}

	// A second request for the same pending pair must fail.
	if _, err := svc.RequestProductAdmin(context.Background(), admin, p2.ID, newUserSub.UserID); !errors.Is(err, ErrRequestAlreadyPending) {
		t.Fatalf("expected ErrRequestAlreadyPending, got %v", err)
	}

	// A plain admin cannot decide.
	if _, err := svc.DecideRequest(context.Background(), admin, req.ID, true); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden for admin deciding, got %v", err)
	}

	decided, err := svc.DecideRequest(context.Background(), superadmin, req.ID, true)
	if err != nil {
		t.Fatalf("decide request: %v", err)
	}
	if decided.Status != models.ProductAdminRequestApproved {
		t.Fatalf("expected approved status, got %q", decided.Status)
	}

	access, err := svc.CheckAccess(context.Background(), newUserSub.UserID, p2.ID)
	if err != nil {
		t.Fatalf("check access after approval: %v", err)
	}
	if access != AccessFull {
		t.Fatalf("expected full access to p2 after approval, got %q", access)
	}
}

func TestProductService_PromoteDirectly_SuperadminOnly(t *testing.T) {
	svc := newTestProductService(t)
	p, err := svc.Create(context.Background(), "Vault")
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	superadmin := shopassign.Caller{SystemRole: models.SystemRoleSuperadmin}
	admin := shopassign.Caller{SystemRole: models.SystemRoleAdmin}

	sub, err := svc.CreateUserAndSubscribe(context.Background(), superadmin, p.ID, CreateUserInput{
		Name: "Direct Subject", Username: "direct-" + p.ID, Email: "direct-" + p.ID + "@example.com", Password: "password123",
	}, models.SystemRoleUser)
	if err != nil {
		t.Fatalf("create subject: %v", err)
	}

	if _, err := svc.PromoteDirectly(context.Background(), admin, p.ID, sub.UserID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden for admin calling PromoteDirectly, got %v", err)
	}

	if _, err := svc.PromoteDirectly(context.Background(), superadmin, p.ID, sub.UserID); err != nil {
		t.Fatalf("promote directly: %v", err)
	}
}
