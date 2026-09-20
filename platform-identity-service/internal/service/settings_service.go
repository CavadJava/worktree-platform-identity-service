package service

import (
	"context"
	"errors"
	"strconv"

	"platform-identity-service/internal/auth"
	"platform-identity-service/internal/models"
	"platform-identity-service/internal/repository"
	"platform-identity-service/internal/service/shopassign"
)

const settingKeyJWTTTLMinutes = "jwt_ttl_minutes"

var ErrInvalidTTL = errors.New("jwt_ttl_minutes must be a positive integer")

// SettingsService is superadmin-only for both reads and writes — these are
// platform-wide operational knobs, not something any logged-in user
// should be able to see or change.
type SettingsService struct {
	repo *repository.SettingsRepository
	jwt  *auth.JWTManager
}

func NewSettingsService(repo *repository.SettingsRepository, jwt *auth.JWTManager) *SettingsService {
	return &SettingsService{repo: repo, jwt: jwt}
}

// GetJWTTTLMinutes reads the current token lifetime setting.
func (s *SettingsService) GetJWTTTLMinutes(ctx context.Context, caller shopassign.Caller) (int, error) {
	if caller.SystemRole != models.SystemRoleSuperadmin {
		return 0, ErrForbidden
	}
	raw, err := s.repo.Get(ctx, settingKeyJWTTTLMinutes)
	if err != nil {
		return 0, err
	}
	minutes, err := strconv.Atoi(raw)
	if err != nil {
		return 0, err
	}
	return minutes, nil
}

// SetJWTTTLMinutes updates the token lifetime setting and immediately
// applies it to the shared JWTManager, so every login from this point on
// (no restart required) uses the new duration. Tokens already issued keep
// whatever lifetime they were minted with — this only affects future
// logins.
func (s *SettingsService) SetJWTTTLMinutes(ctx context.Context, caller shopassign.Caller, minutes int) error {
	if caller.SystemRole != models.SystemRoleSuperadmin {
		return ErrForbidden
	}
	if minutes <= 0 {
		return ErrInvalidTTL
	}
	if err := s.repo.Set(ctx, settingKeyJWTTTLMinutes, strconv.Itoa(minutes)); err != nil {
		return err
	}
	s.jwt.SetTTL(minutes)
	return nil
}
