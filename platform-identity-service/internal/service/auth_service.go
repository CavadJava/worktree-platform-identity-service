package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"platform-identity-service/internal/auth"
	"platform-identity-service/internal/models"
	"platform-identity-service/internal/repository"
)

const systemRoleIDUser int16 = 3

var (
	ErrUserNotFound       = repository.ErrUserNotFound
	ErrUsernameTaken      = repository.ErrUsernameTaken
	ErrEmailTaken         = repository.ErrEmailTaken
	ErrInvalidCredentials = errors.New("invalid username/email or password")
)

type AuthService struct {
	userRepo *repository.UserRepository
	jwt      *auth.JWTManager
}

func NewAuthService(userRepo *repository.UserRepository, jwt *auth.JWTManager) *AuthService {
	return &AuthService{userRepo: userRepo, jwt: jwt}
}

type RegisterInput struct {
	Name     string
	Username string
	Email    string
	Password string
}

// Register creates a Teslahubs-wide account. Every registered user starts
// as system role 'user' with no shop memberships — shop membership is
// admin-granted afterward via ShopMembershipService, not part of signup.
func (s *AuthService) Register(ctx context.Context, in RegisterInput) (*models.User, error) {
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	u := &models.User{
		ID:           uuid.NewString(),
		Name:         in.Name,
		Username:     in.Username,
		Email:        in.Email,
		PasswordHash: hash,
		SystemRoleID: systemRoleIDUser,
		Status:       models.UserStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.userRepo.Create(ctx, u); err != nil {
		return nil, err
	}

	return s.userRepo.GetByID(ctx, u.ID)
}

type LoginInput struct {
	Identifier string
	Password   string
}

func (s *AuthService) Login(ctx context.Context, in LoginInput) (string, time.Time, error) {
	u, err := s.userRepo.GetByUsernameOrEmail(ctx, in.Identifier)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return "", time.Time{}, ErrInvalidCredentials
		}
		return "", time.Time{}, err
	}

	if !auth.CheckPassword(u.PasswordHash, in.Password) {
		return "", time.Time{}, ErrInvalidCredentials
	}

	if u.Status != models.UserStatusActive {
		return "", time.Time{}, ErrInvalidCredentials
	}

	return s.jwt.Generate(u.ID, u.SystemRoleName)
}
