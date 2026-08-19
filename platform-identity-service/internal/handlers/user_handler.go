package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"platform-identity-service/internal/middleware"
	"platform-identity-service/internal/models"
	"platform-identity-service/internal/repository"
	"platform-identity-service/internal/service"
)

type UserHandler struct {
	svc           *service.UserService
	membershipSvc *service.ShopMembershipService
}

func NewUserHandler(svc *service.UserService, membershipSvc *service.ShopMembershipService) *UserHandler {
	return &UserHandler{svc: svc, membershipSvc: membershipSvc}
}

// Get godoc
// @Summary      Get a user
// @Description  Özünü, ya da (superadmin olarsa) istənilən useri görə bilər.
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "User ID"
// @Success      200 {object} userResponse
// @Failure      403 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /users/{id} [get]
func (h *UserHandler) Get(w http.ResponseWriter, r *http.Request) {
	caller, ok := middleware.CallerFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	u, err := h.svc.Get(r.Context(), caller, id)
	writeUserOrError(w, u, err)
}

// ListAll godoc
// @Summary      List every Teslahubs user
// @Description  Yalnız superadmin çağıra bilər.
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} userResponse
// @Failure      403 {object} map[string]string
// @Router       /users [get]
func (h *UserHandler) ListAll(w http.ResponseWriter, r *http.Request) {
	caller, ok := middleware.CallerFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	users, err := h.svc.ListAll(r.Context(), caller)
	if err != nil {
		writeUserServiceError(w, err)
		return
	}

	response := make([]userResponse, len(users))
	for i, u := range users {
		response[i] = userResponse{ID: u.ID, Name: u.Name, Username: u.Username, Email: u.Email, SystemRole: u.SystemRoleName, Status: u.Status}
	}
	writeJSON(w, http.StatusOK, response)
}

type setSystemRoleRequest struct {
	SystemRole string `json:"system_role"`
}

// SetSystemRole godoc
// @Summary      Change a user's system role
// @Description  Yalnız superadmin çağıra bilər.
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "User ID"
// @Param        request body setSystemRoleRequest true "Role payload"
// @Success      200 {object} userResponse
// @Failure      403 {object} map[string]string
// @Router       /users/{id}/system-role [post]
func (h *UserHandler) SetSystemRole(w http.ResponseWriter, r *http.Request) {
	caller, ok := middleware.CallerFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req setSystemRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.SystemRole != models.SystemRoleSuperadmin && req.SystemRole != models.SystemRoleAdmin && req.SystemRole != models.SystemRoleUser {
		writeError(w, http.StatusBadRequest, "system_role must be 'superadmin', 'admin', or 'user'")
		return
	}

	id := chi.URLParam(r, "id")
	u, err := h.svc.SetSystemRole(r.Context(), caller, id, req.SystemRole)
	writeUserOrError(w, u, err)
}

type setStatusRequest struct {
	Status string `json:"status"`
}

// SetStatus godoc
// @Summary      Activate or deactivate a user
// @Description  Yalnız superadmin çağıra bilər.
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "User ID"
// @Param        request body setStatusRequest true "Status payload"
// @Success      200 {object} userResponse
// @Failure      403 {object} map[string]string
// @Router       /users/{id}/status [post]
func (h *UserHandler) SetStatus(w http.ResponseWriter, r *http.Request) {
	caller, ok := middleware.CallerFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req setStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Status != models.UserStatusActive && req.Status != models.UserStatusInActive {
		writeError(w, http.StatusBadRequest, "status must be 'ACTIVE' or 'IN_ACTIVE'")
		return
	}

	id := chi.URLParam(r, "id")
	u, err := h.svc.SetStatus(r.Context(), caller, id, req.Status)
	writeUserOrError(w, u, err)
}

type updateProfileRequest struct {
	Name     *string `json:"name"`
	Email    *string `json:"email"`
	Password *string `json:"password"`
}

// UpdateProfile godoc
// @Summary      Edit a user's name/email/password
// @Description  Özünü, ya da (superadmin olarsa) istənilən useri redaktə edə bilər. Bütün sahələr könüllüdür.
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "User ID"
// @Param        request body updateProfileRequest true "Profile payload"
// @Success      200 {object} userResponse
// @Failure      403 {object} map[string]string
// @Router       /users/{id}/profile [post]
func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	caller, ok := middleware.CallerFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req updateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	id := chi.URLParam(r, "id")
	u, err := h.svc.UpdateProfile(r.Context(), caller, id, service.ProfileUpdate{
		Name: req.Name, Email: req.Email, Password: req.Password,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUsernameTaken):
			writeError(w, http.StatusConflict, "username already registered")
		case errors.Is(err, service.ErrEmailTaken):
			writeError(w, http.StatusConflict, "email already registered")
		default:
			writeUserOrError(w, nil, err)
		}
		return
	}
	writeJSON(w, http.StatusOK, userResponse{
		ID: u.ID, Name: u.Name, Username: u.Username, Email: u.Email, SystemRole: u.SystemRoleName, Status: u.Status,
	})
}

// ListMyShops godoc
// @Summary      List the shops a user belongs to
// @Description  Özünü, ya da (superadmin olarsa) istənilən useri görə bilər.
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "User ID"
// @Success      200 {array} memberResponse
// @Failure      403 {object} map[string]string
// @Router       /users/{id}/shops [get]
func (h *UserHandler) ListMyShops(w http.ResponseWriter, r *http.Request) {
	caller, ok := middleware.CallerFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	userID := chi.URLParam(r, "id")
	if caller.UserID != userID && caller.SystemRole != models.SystemRoleSuperadmin {
		writeError(w, http.StatusForbidden, "self or superadmin required")
		return
	}

	memberships, err := h.membershipSvc.ListMyShops(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list shops")
		return
	}

	response := make([]memberResponse, len(memberships))
	for i := range memberships {
		response[i] = toMemberResponse(&memberships[i])
	}
	writeJSON(w, http.StatusOK, response)
}

func writeUserServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrForbidden):
		writeError(w, http.StatusForbidden, "superadmin role required")
	case errors.Is(err, service.ErrInvalidSystemRole), errors.Is(err, service.ErrInvalidStatus):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "failed to process request")
	}
}

func writeUserOrError(w http.ResponseWriter, u *models.User, err error) {
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrUserNotFound):
			writeError(w, http.StatusNotFound, "user not found")
		default:
			writeUserServiceError(w, err)
		}
		return
	}
	writeJSON(w, http.StatusOK, userResponse{
		ID: u.ID, Name: u.Name, Username: u.Username, Email: u.Email, SystemRole: u.SystemRoleName, Status: u.Status,
	})
}
