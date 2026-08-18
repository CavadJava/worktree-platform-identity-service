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
	models.RoleUser:  roleIDUser,
	models.RoleAdmin: roleIDAdmin,
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

// ListAll returns every user across every project. Unlike the other
// UserService methods, this check does not go through the injected
// RoleAssigner — there is no single target project to test CanAssign
// against, since "give me everything" is a different shape of question
// than "can I act on this one project/user."
func (s *UserService) ListAll(ctx context.Context, caller roleassign.Caller) ([]models.User, error) {
	if caller.Role != models.RoleSuperadmin {
		return nil, ErrForbidden
	}
	return s.repo.ListAll(ctx)
}

// ListByProject returns every user in projectID. Caller must be an admin
// within that same project — reuses the same RoleAssigner check as SetRole,
// with the project itself as the target (UserID is irrelevant to CanAssign).
func (s *UserService) ListByProject(ctx context.Context, caller roleassign.Caller, projectID string) ([]models.User, error) {
	if !s.assigner.CanAssign(caller, roleassign.Target{ProjectID: projectID}) {
		return nil, ErrForbidden
	}
	return s.repo.ListByProject(ctx, projectID)
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
