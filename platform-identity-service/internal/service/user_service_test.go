package service

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"platform-identity-service/internal/repository"
	"platform-identity-service/internal/service/roleassign"
)

func newTestUserService(t *testing.T) (*UserService, *AuthService, *ProjectService) {
	t.Helper()
	db := testDB(t)
	projectRepo := repository.NewProjectRepository(db)
	userRepo := repository.NewUserRepository(db)
	authSvc, projectSvc := newAuthAndProjectServiceFromRepos(t, projectRepo, userRepo)
	userSvc := NewUserService(userRepo, roleassign.NewSameProjectAdmin())
	return userSvc, authSvc, projectSvc
}

func TestUserService_SetRole_SameProjectAdminSucceeds(t *testing.T) {
	userSvc, authSvc, projectSvc := newTestUserService(t)

	p, _ := projectSvc.Create(context.Background(), "SetRole Success "+uuid.NewString())
	admin, err := authSvc.Register(context.Background(), RegisterInput{
		ProjectID: p.ID, Name: "Admin", Username: "admin-" + uuid.NewString(),
		Email: uuid.NewString() + "@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("register admin failed: %v", err)
	}
	member, err := authSvc.Register(context.Background(), RegisterInput{
		ProjectID: p.ID, Name: "Member", Username: "member-" + uuid.NewString(),
		Email: uuid.NewString() + "@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("register member failed: %v", err)
	}

	caller := roleassign.Caller{UserID: admin.ID, ProjectID: p.ID, Role: admin.RoleName}
	updated, err := userSvc.SetRole(context.Background(), caller, member.ID, "admin")
	if err != nil {
		t.Fatalf("SetRole failed: %v", err)
	}
	if updated.RoleName != "admin" {
		t.Errorf("expected member promoted to admin, got %q", updated.RoleName)
	}
}

func TestUserService_Get_NonAdminCannotReadAnotherUser(t *testing.T) {
	userSvc, authSvc, projectSvc := newTestUserService(t)

	p, _ := projectSvc.Create(context.Background(), "Get Forbidden "+uuid.NewString())
	admin, err := authSvc.Register(context.Background(), RegisterInput{
		ProjectID: p.ID, Name: "Admin", Username: "admin-" + uuid.NewString(),
		Email: uuid.NewString() + "@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("register admin failed: %v", err)
	}
	member, err := authSvc.Register(context.Background(), RegisterInput{
		ProjectID: p.ID, Name: "Member", Username: "member-" + uuid.NewString(),
		Email: uuid.NewString() + "@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("register member failed: %v", err)
	}

	callerForSecondUser := roleassign.Caller{UserID: member.ID, ProjectID: p.ID, Role: member.RoleName}
	_, err = userSvc.Get(context.Background(), callerForSecondUser, admin.ID)
	if err != ErrForbidden {
		t.Errorf("expected ErrForbidden for non-admin reading another user, got %v", err)
	}
}

func TestUserService_SetRole_InvalidRoleNameRejected(t *testing.T) {
	userSvc, authSvc, projectSvc := newTestUserService(t)

	p, _ := projectSvc.Create(context.Background(), "Invalid Role "+uuid.NewString())
	admin, err := authSvc.Register(context.Background(), RegisterInput{
		ProjectID: p.ID, Name: "Admin", Username: "admin-" + uuid.NewString(),
		Email: uuid.NewString() + "@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("register admin failed: %v", err)
	}
	member, err := authSvc.Register(context.Background(), RegisterInput{
		ProjectID: p.ID, Name: "Member", Username: "member-" + uuid.NewString(),
		Email: uuid.NewString() + "@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("register member failed: %v", err)
	}

	caller := roleassign.Caller{UserID: admin.ID, ProjectID: p.ID, Role: admin.RoleName}
	_, err = userSvc.SetRole(context.Background(), caller, member.ID, "superadmin")
	if err == nil {
		t.Fatal("expected error for invalid role name, got nil")
	}
}

func TestUserService_ListByProject_SameProjectAdminSucceeds(t *testing.T) {
	userSvc, authSvc, projectSvc := newTestUserService(t)

	p, _ := projectSvc.Create(context.Background(), "ListByProject Success "+uuid.NewString())
	admin, err := authSvc.Register(context.Background(), RegisterInput{
		ProjectID: p.ID, Name: "Admin", Username: "admin-" + uuid.NewString(),
		Email: uuid.NewString() + "@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("register admin failed: %v", err)
	}
	_, err = authSvc.Register(context.Background(), RegisterInput{
		ProjectID: p.ID, Name: "Member", Username: "member-" + uuid.NewString(),
		Email: uuid.NewString() + "@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("register member failed: %v", err)
	}

	caller := roleassign.Caller{UserID: admin.ID, ProjectID: p.ID, Role: admin.RoleName}
	users, err := userSvc.ListByProject(context.Background(), caller, p.ID)
	if err != nil {
		t.Fatalf("ListByProject failed: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users (admin + member), got %d", len(users))
	}
}

func TestUserService_ListByProject_CrossProjectForbidden(t *testing.T) {
	userSvc, authSvc, projectSvc := newTestUserService(t)

	p1, _ := projectSvc.Create(context.Background(), "ListByProject Cross A "+uuid.NewString())
	p2, _ := projectSvc.Create(context.Background(), "ListByProject Cross B "+uuid.NewString())

	admin1, err := authSvc.Register(context.Background(), RegisterInput{
		ProjectID: p1.ID, Name: "Admin1", Username: "admin1-" + uuid.NewString(),
		Email: uuid.NewString() + "@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("register admin1 failed: %v", err)
	}

	caller := roleassign.Caller{UserID: admin1.ID, ProjectID: p1.ID, Role: admin1.RoleName}
	_, err = userSvc.ListByProject(context.Background(), caller, p2.ID)
	if err != ErrForbidden {
		t.Errorf("expected ErrForbidden for cross-project list, got %v", err)
	}
}

func TestUserService_SetRole_CrossProjectForbidden(t *testing.T) {
	userSvc, authSvc, projectSvc := newTestUserService(t)

	p1, _ := projectSvc.Create(context.Background(), "Cross Project A "+uuid.NewString())
	p2, _ := projectSvc.Create(context.Background(), "Cross Project B "+uuid.NewString())

	admin1, err := authSvc.Register(context.Background(), RegisterInput{
		ProjectID: p1.ID, Name: "Admin1", Username: "admin1-" + uuid.NewString(),
		Email: uuid.NewString() + "@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("register admin1 failed: %v", err)
	}
	member2, err := authSvc.Register(context.Background(), RegisterInput{
		ProjectID: p2.ID, Name: "Member2", Username: "member2-" + uuid.NewString(),
		Email: uuid.NewString() + "@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("register member2 failed: %v", err)
	}

	caller := roleassign.Caller{UserID: admin1.ID, ProjectID: p1.ID, Role: admin1.RoleName}
	_, err = userSvc.SetRole(context.Background(), caller, member2.ID, "admin")
	if err != ErrForbidden {
		t.Errorf("expected ErrForbidden for cross-project assignment, got %v", err)
	}
}
