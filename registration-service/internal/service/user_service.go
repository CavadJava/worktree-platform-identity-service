package service

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"

	"registration-service/internal/auth"
	"registration-service/internal/models"
	"registration-service/internal/repository"
	"registration-service/internal/roles"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrEmailTaken         = repository.ErrEmailTaken
	ErrUserNotFound       = repository.ErrUserNotFound
)

type notifier interface {
	SendWelcome(ctx context.Context, email, fullName string) error
}

type UserService struct {
	repo     *repository.UserRepository
	jwt      *auth.JWTManager
	notifier notifier
}

func NewUserService(repo *repository.UserRepository, jwt *auth.JWTManager, notifier notifier) *UserService {
	return &UserService{repo: repo, jwt: jwt, notifier: notifier}
}

type RegisterInput struct {
	Email    string
	Password string
	FullName string
	Phone    string
}

func (s *UserService) Register(ctx context.Context, in RegisterInput) (*models.User, error) {
	email := normalizeEmail(in.Email)

	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	u := &models.User{
		ID:            uuid.NewString(),
		Email:         email,
		PasswordHash:  hash,
		FullName:      in.FullName,
		Phone:         in.Phone,
		Role:          roles.RoleUser,
		ShopID:        nil,
		ShopRoleLevel: roles.ShopLevelNone,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.repo.Create(ctx, u); err != nil {
		return nil, err
	}

	// Notification failure must not fail registration — the user is already created.
	if err := s.notifier.SendWelcome(ctx, u.Email, u.FullName); err != nil {
		log.Printf("warning: failed to send welcome notification for %s: %v", u.Email, err)
	}

	return u, nil
}

type LoginResult struct {
	Token     string
	ExpiresAt time.Time
	User      *models.User
}

func (s *UserService) Login(ctx context.Context, email, password string) (*LoginResult, error) {
	u, err := s.repo.FindByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if !auth.CheckPassword(u.PasswordHash, password) {
		return nil, ErrInvalidCredentials
	}

	token, expiresAt, err := s.jwt.Generate(u.ID, u.Role, u.ShopID, u.ShopRoleLevel)
	if err != nil {
		return nil, err
	}

	return &LoginResult{Token: token, ExpiresAt: expiresAt, User: u}, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
