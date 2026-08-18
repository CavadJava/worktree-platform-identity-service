package service

import (
	"context"
	"errors"

	"platform-identity-service/internal/models"
	"platform-identity-service/internal/repository"
	"platform-identity-service/internal/service/roleassign"
)

var ErrForbidden = errors.New("forbidden")

var roleNameToID = map[string]int16{
	"user":  roleIDUser,
	"admin": roleIDAdmin,
}

type UserService struct {
	repo     *repository.UserRepository
	assigner roleassign.RoleAssigner
}

func NewUserService(repo *repository.UserRepository, assigner roleassign.RoleAssigner) *UserService {
	return &UserService{repo: repo, assigner: assigner}
}

// Get returns userID's record if caller is that same user, or an admin
// within the target's own project.
func (s *UserService) Get(ctx context.Context, caller roleassign.Caller, userID string) (*models.User, error) {
	target, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	targetProjectID := ""
	if target.ProjectID != nil {
		targetProjectID = *target.ProjectID
	}

	isSelf := caller.UserID == userID
	isSameProjectAdmin := s.assigner.CanAssign(caller, roleassign.Target{UserID: userID, ProjectID: targetProjectID})
	if !isSelf && !isSameProjectAdmin {
		return nil, ErrForbidden
	}

	return target, nil
}

func (s *UserService) SetRole(ctx context.Context, caller roleassign.Caller, targetUserID, newRoleName string) (*models.User, error) {
	target, err := s.repo.GetByID(ctx, targetUserID)
	if err != nil {
		return nil, err
	}

	targetProjectID := ""
	if target.ProjectID != nil {
		targetProjectID = *target.ProjectID
	}

	if !s.assigner.CanAssign(caller, roleassign.Target{UserID: targetUserID, ProjectID: targetProjectID}) {
		return nil, ErrForbidden
	}

	roleID, ok := roleNameToID[newRoleName]
	if !ok {
		return nil, errors.New("invalid role name: must be 'user' or 'admin'")
	}

	if err := s.repo.SetRole(ctx, targetUserID, roleID); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, targetUserID)
}
