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

const (
	roleIDUser  int16 = 1
	roleIDAdmin int16 = 2
)

var (
	ErrUserNotFound       = repository.ErrUserNotFound
	ErrUsernameTaken      = repository.ErrUsernameTaken
	ErrEmailTaken         = repository.ErrEmailTaken
	ErrInvalidCredentials = errors.New("invalid username/email or password")
)

type AuthService struct {
	projectRepo *repository.ProjectRepository
	userRepo    *repository.UserRepository
	jwt         *auth.JWTManager
}

func NewAuthService(projectRepo *repository.ProjectRepository, userRepo *repository.UserRepository, jwt *auth.JWTManager) *AuthService {
	return &AuthService{projectRepo: projectRepo, userRepo: userRepo, jwt: jwt}
}

type RegisterInput struct {
	ProjectID string
	Name      string
	Username  string
	Email     string
	Password  string
}

// Register creates a user under the given project. The first user ever
// registered for a project becomes admin; every later one becomes user.
// CountByProject + Create both run against the same *sql.DB without an
// explicit transaction here — acceptable for this practice project's
// traffic level, but the race window (two concurrent first registrations
// both seeing count==0) is a known, documented limitation, not an oversight.
func (s *AuthService) Register(ctx context.Context, in RegisterInput) (*models.User, error) {
	if _, err := s.projectRepo.GetByID(ctx, in.ProjectID); err != nil {
		return nil, err
	}

	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return nil, err
	}

	count, err := s.userRepo.CountByProject(ctx, in.ProjectID)
	if err != nil {
		return nil, err
	}
	roleID := roleIDUser
	if count == 0 {
		roleID = roleIDAdmin
	}

	now := time.Now().UTC()
	projectID := in.ProjectID
	u := &models.User{
		ID:           uuid.NewString(),
		Name:         in.Name,
		Username:     in.Username,
		Email:        in.Email,
		PasswordHash: hash,
		ProjectID:    &projectID,
		RoleID:       &roleID,
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

	projectID := ""
	if u.ProjectID != nil {
		projectID = *u.ProjectID
	}
	return s.jwt.Generate(u.ID, projectID, u.RoleName)
}
