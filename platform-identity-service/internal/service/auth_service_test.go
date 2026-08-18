package service

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"platform-identity-service/internal/auth"
	"platform-identity-service/internal/repository"
)

func newTestAuthService(t *testing.T) (*AuthService, *ProjectService) {
	t.Helper()
	db := testDB(t)
	projectRepo := repository.NewProjectRepository(db)
	userRepo := repository.NewUserRepository(db)
	jwt := auth.NewJWTManager("test-secret", 60)
	return NewAuthService(projectRepo, userRepo, jwt), NewProjectService(projectRepo)
}

func TestAuthService_Register_FirstUserBecomesAdmin(t *testing.T) {
	authSvc, projectSvc := newTestAuthService(t)

	p, err := projectSvc.Create(context.Background(), "First-Admin Test "+uuid.NewString())
	if err != nil {
		t.Fatalf("Create project failed: %v", err)
	}

	first, err := authSvc.Register(context.Background(), RegisterInput{
		ProjectID: p.ID, Name: "First", Username: "first-" + uuid.NewString(),
		Email: uuid.NewString() + "@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("Register (first) failed: %v", err)
	}
	if first.RoleName != "admin" {
		t.Errorf("expected first registered user to be 'admin', got %q", first.RoleName)
	}

	second, err := authSvc.Register(context.Background(), RegisterInput{
		ProjectID: p.ID, Name: "Second", Username: "second-" + uuid.NewString(),
		Email: uuid.NewString() + "@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("Register (second) failed: %v", err)
	}
	if second.RoleName != "user" {
		t.Errorf("expected second registered user to be 'user', got %q", second.RoleName)
	}
}

func TestAuthService_Register_DuplicateUsername(t *testing.T) {
	authSvc, projectSvc := newTestAuthService(t)
	p, _ := projectSvc.Create(context.Background(), "Dup Test "+uuid.NewString())

	username := "dup-" + uuid.NewString()
	_, err := authSvc.Register(context.Background(), RegisterInput{
		ProjectID: p.ID, Name: "A", Username: username, Email: uuid.NewString() + "@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("first Register failed: %v", err)
	}

	_, err = authSvc.Register(context.Background(), RegisterInput{
		ProjectID: p.ID, Name: "B", Username: username, Email: uuid.NewString() + "@example.com", Password: "password123",
	})
	if err != ErrUsernameTaken {
		t.Errorf("expected ErrUsernameTaken, got %v", err)
	}
}

func TestAuthService_Login_Success(t *testing.T) {
	authSvc, projectSvc := newTestAuthService(t)
	p, _ := projectSvc.Create(context.Background(), "Login Test "+uuid.NewString())

	username := "login-" + uuid.NewString()
	_, err := authSvc.Register(context.Background(), RegisterInput{
		ProjectID: p.ID, Name: "Login User", Username: username,
		Email: uuid.NewString() + "@example.com", Password: "correct-password",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	token, _, err := authSvc.Login(context.Background(), LoginInput{Identifier: username, Password: "correct-password"})
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if token == "" {
		t.Error("expected non-empty token")
	}
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	authSvc, projectSvc := newTestAuthService(t)
	p, _ := projectSvc.Create(context.Background(), "Login Fail Test "+uuid.NewString())

	username := "loginfail-" + uuid.NewString()
	_, err := authSvc.Register(context.Background(), RegisterInput{
		ProjectID: p.ID, Name: "User", Username: username,
		Email: uuid.NewString() + "@example.com", Password: "correct-password",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	_, _, err = authSvc.Login(context.Background(), LoginInput{Identifier: username, Password: "wrong-password"})
	if err != ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}
