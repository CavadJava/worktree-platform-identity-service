package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"platform-identity-service/internal/models"
	"platform-identity-service/internal/repository"
	"platform-identity-service/internal/service/shopassign"
)

func newTestUserService(t *testing.T) (*UserService, *AuthService) {
	t.Helper()
	db := testDB(t)
	userRepo := repository.NewUserRepository(db)
	jwtMgr := newTestJWTManager()
	authSvc := NewAuthService(userRepo, jwtMgr)
	userSvc := NewUserService(userRepo)
	return userSvc, authSvc
}

func TestUserService_Get_Self(t *testing.T) {
	userSvc, authSvc := newTestUserService(t)

	u, err := authSvc.Register(context.Background(), RegisterInput{
		Name: "Self", Username: "self-" + uuid.NewString(), Email: uuid.NewString() + "@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	caller := shopassign.Caller{UserID: u.ID, SystemRole: models.SystemRoleUser}
	got, err := userSvc.Get(context.Background(), caller, u.ID)
	if err != nil {
		t.Fatalf("Get self failed: %v", err)
	}
	if got.ID != u.ID {
		t.Errorf("expected id %q, got %q", u.ID, got.ID)
	}
}

func TestUserService_Get_ForbiddenForOtherPlainUser(t *testing.T) {
	userSvc, authSvc := newTestUserService(t)

	caller, err := authSvc.Register(context.Background(), RegisterInput{
		Name: "Caller", Username: "caller-" + uuid.NewString(), Email: uuid.NewString() + "@example.com", Password: "password123",
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

	callerClaims := shopassign.Caller{UserID: caller.ID, SystemRole: models.SystemRoleUser}
	_, err = userSvc.Get(context.Background(), callerClaims, target.ID)
	if err != ErrForbidden {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestUserService_Get_SuperadminCanViewAnyone(t *testing.T) {
	userSvc, authSvc := newTestUserService(t)

	target, err := authSvc.Register(context.Background(), RegisterInput{
		Name: "Target", Username: "target-" + uuid.NewString(), Email: uuid.NewString() + "@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("register target failed: %v", err)
	}

	superadminCaller := shopassign.Caller{UserID: "superadmin-id", SystemRole: models.SystemRoleSuperadmin}
	got, err := userSvc.Get(context.Background(), superadminCaller, target.ID)
	if err != nil {
		t.Fatalf("Get by superadmin failed: %v", err)
	}
	if got.ID != target.ID {
		t.Errorf("expected id %q, got %q", target.ID, got.ID)
	}
}

func TestUserService_ListAll_SuperadminOnly(t *testing.T) {
	userSvc, authSvc := newTestUserService(t)

	_, err := authSvc.Register(context.Background(), RegisterInput{
		Name: "Someone", Username: "someone-" + uuid.NewString(), Email: uuid.NewString() + "@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	superadminCaller := shopassign.Caller{UserID: "superadmin-id", SystemRole: models.SystemRoleSuperadmin}
	users, err := userSvc.ListAll(context.Background(), superadminCaller)
	if err != nil {
		t.Fatalf("ListAll by superadmin failed: %v", err)
	}
	if len(users) == 0 {
		t.Error("expected at least one user")
	}

	plainCaller := shopassign.Caller{UserID: "someone-id", SystemRole: models.SystemRoleUser}
	_, err = userSvc.ListAll(context.Background(), plainCaller)
	if err != ErrForbidden {
		t.Errorf("expected ErrForbidden for plain user, got %v", err)
	}
}

func TestUserService_SetSystemRole_SuperadminOnly(t *testing.T) {
	userSvc, authSvc := newTestUserService(t)

	target, err := authSvc.Register(context.Background(), RegisterInput{
		Name: "Target", Username: "target-" + uuid.NewString(), Email: uuid.NewString() + "@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("register target failed: %v", err)
	}

	superadminCaller := shopassign.Caller{UserID: "superadmin-id", SystemRole: models.SystemRoleSuperadmin}
	updated, err := userSvc.SetSystemRole(context.Background(), superadminCaller, target.ID, models.SystemRoleAdmin)
	if err != nil {
		t.Fatalf("SetSystemRole failed: %v", err)
	}
	if updated.SystemRoleName != models.SystemRoleAdmin {
		t.Errorf("expected 'admin', got %q", updated.SystemRoleName)
	}

	plainCaller := shopassign.Caller{UserID: target.ID, SystemRole: models.SystemRoleAdmin}
	_, err = userSvc.SetSystemRole(context.Background(), plainCaller, target.ID, models.SystemRoleSuperadmin)
	if err != ErrForbidden {
		t.Errorf("expected ErrForbidden for non-superadmin, got %v", err)
	}
}

func TestUserService_SetStatus_SuperadminOnly(t *testing.T) {
	userSvc, authSvc := newTestUserService(t)

	target, err := authSvc.Register(context.Background(), RegisterInput{
		Name: "Target", Username: "target-" + uuid.NewString(), Email: uuid.NewString() + "@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("register target failed: %v", err)
	}

	superadminCaller := shopassign.Caller{UserID: "superadmin-id", SystemRole: models.SystemRoleSuperadmin}
	updated, err := userSvc.SetStatus(context.Background(), superadminCaller, target.ID, models.UserStatusInActive)
	if err != nil {
		t.Fatalf("SetStatus failed: %v", err)
	}
	if updated.Status != models.UserStatusInActive {
		t.Errorf("expected IN_ACTIVE, got %q", updated.Status)
	}
}

func TestUserService_UpdateProfile_Self(t *testing.T) {
	userSvc, authSvc := newTestUserService(t)

	u, err := authSvc.Register(context.Background(), RegisterInput{
		Name: "Original Name", Username: "profile-self-" + uuid.NewString(), Email: uuid.NewString() + "@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	caller := shopassign.Caller{UserID: u.ID, SystemRole: models.SystemRoleUser}
	newName := "New Name"
	updated, err := userSvc.UpdateProfile(context.Background(), caller, u.ID, ProfileUpdate{Name: &newName})
	if err != nil {
		t.Fatalf("UpdateProfile failed: %v", err)
	}
	if updated.Name != newName {
		t.Errorf("expected name %q, got %q", newName, updated.Name)
	}
}

func TestUserService_UpdateProfile_ForbiddenForOtherPlainUser(t *testing.T) {
	userSvc, authSvc := newTestUserService(t)

	caller, err := authSvc.Register(context.Background(), RegisterInput{
		Name: "Caller", Username: "profile-caller-" + uuid.NewString(), Email: uuid.NewString() + "@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("register caller failed: %v", err)
	}
	target, err := authSvc.Register(context.Background(), RegisterInput{
		Name: "Target", Username: "profile-forbidden-target-" + uuid.NewString(), Email: uuid.NewString() + "@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("register target failed: %v", err)
	}

	callerClaims := shopassign.Caller{UserID: caller.ID, SystemRole: models.SystemRoleUser}
	newName := "Hijacked"
	_, err = userSvc.UpdateProfile(context.Background(), callerClaims, target.ID, ProfileUpdate{Name: &newName})
	if err != ErrForbidden {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestUserService_UpdateProfile_SuperadminCanEditAnyone(t *testing.T) {
	userSvc, authSvc := newTestUserService(t)

	target, err := authSvc.Register(context.Background(), RegisterInput{
		Name: "Target", Username: "profile-superadmin-target-" + uuid.NewString(), Email: uuid.NewString() + "@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("register target failed: %v", err)
	}

	superadminCaller := shopassign.Caller{UserID: "superadmin-id", SystemRole: models.SystemRoleSuperadmin}
	newEmail := uuid.NewString() + "@example.com"
	updated, err := userSvc.UpdateProfile(context.Background(), superadminCaller, target.ID, ProfileUpdate{Email: &newEmail})
	if err != nil {
		t.Fatalf("UpdateProfile by superadmin failed: %v", err)
	}
	if updated.Email != newEmail {
		t.Errorf("expected email %q, got %q", newEmail, updated.Email)
	}
}

func TestUserService_ListBasic_AllowsAdminNotJustSuperadmin(t *testing.T) {
	userSvc, authSvc := newTestUserService(t)

	_, err := authSvc.Register(context.Background(), RegisterInput{
		Name: "Someone", Username: "someone-" + uuid.NewString(), Email: uuid.NewString() + "@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	admin := shopassign.Caller{SystemRole: models.SystemRoleAdmin}
	users, err := userSvc.ListBasic(context.Background(), admin)
	if err != nil {
		t.Fatalf("expected admin to list basic users, got %v", err)
	}
	_ = users

	plainUser := shopassign.Caller{SystemRole: models.SystemRoleUser}
	if _, err := userSvc.ListBasic(context.Background(), plainUser); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden for plain user, got %v", err)
	}
}
