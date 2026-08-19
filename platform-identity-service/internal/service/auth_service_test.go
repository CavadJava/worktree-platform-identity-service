package service

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"platform-identity-service/internal/models"
	"platform-identity-service/internal/repository"
)

func TestAuthService_Register_AlwaysCreatesPlainUser(t *testing.T) {
	authSvc := newTestAuthService(t)

	u, err := authSvc.Register(context.Background(), RegisterInput{
		Name: "First", Username: "first-" + uuid.NewString(),
		Email: uuid.NewString() + "@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if u.SystemRoleName != models.SystemRoleUser {
		t.Errorf("expected system role 'user', got %q", u.SystemRoleName)
	}
	if u.Status != models.UserStatusActive {
		t.Errorf("expected status ACTIVE, got %q", u.Status)
	}

	// A second registration also becomes a plain user — no more
	// first-user-becomes-admin behavior, since registration is no longer
	// shop-scoped.
	u2, err := authSvc.Register(context.Background(), RegisterInput{
		Name: "Second", Username: "second-" + uuid.NewString(),
		Email: uuid.NewString() + "@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("second Register failed: %v", err)
	}
	if u2.SystemRoleName != models.SystemRoleUser {
		t.Errorf("expected second user's system role 'user', got %q", u2.SystemRoleName)
	}
}

func TestAuthService_Register_DuplicateUsername(t *testing.T) {
	authSvc := newTestAuthService(t)

	username := "dup-" + uuid.NewString()
	_, err := authSvc.Register(context.Background(), RegisterInput{
		Name: "A", Username: username, Email: uuid.NewString() + "@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("first Register failed: %v", err)
	}

	_, err = authSvc.Register(context.Background(), RegisterInput{
		Name: "B", Username: username, Email: uuid.NewString() + "@example.com", Password: "password123",
	})
	if err != ErrUsernameTaken {
		t.Errorf("expected ErrUsernameTaken, got %v", err)
	}
}

func TestAuthService_Login_Success(t *testing.T) {
	authSvc := newTestAuthService(t)

	username := "login-" + uuid.NewString()
	_, err := authSvc.Register(context.Background(), RegisterInput{
		Name: "Login User", Username: username, Email: uuid.NewString() + "@example.com", Password: "correct-password",
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
	authSvc := newTestAuthService(t)

	username := "loginfail-" + uuid.NewString()
	_, err := authSvc.Register(context.Background(), RegisterInput{
		Name: "User", Username: username, Email: uuid.NewString() + "@example.com", Password: "correct-password",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	_, _, err = authSvc.Login(context.Background(), LoginInput{Identifier: username, Password: "wrong-password"})
	if err != ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthService_Login_InactiveUser(t *testing.T) {
	db := testDB(t)
	userRepo := repository.NewUserRepository(db)
	jwt := newTestJWTManager()
	authSvc := NewAuthService(userRepo, jwt)

	username := "inactive-" + uuid.NewString()
	u, err := authSvc.Register(context.Background(), RegisterInput{
		Name: "Inactive User", Username: username, Email: uuid.NewString() + "@example.com", Password: "correct-password",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if err := userRepo.SetStatus(context.Background(), u.ID, models.UserStatusInActive); err != nil {
		t.Fatalf("SetStatus failed: %v", err)
	}

	_, _, err = authSvc.Login(context.Background(), LoginInput{Identifier: username, Password: "correct-password"})
	if err != ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials for IN_ACTIVE user, got %v", err)
	}
}
