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
	svc *service.UserService
}

func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

// Get godoc
// @Summary      Get a user
// @Description  Özünü, ya da (admin rolunda olarsa) öz layihəsindəki istənilən useri görə bilər.
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

type setRoleRequest struct {
	Role string `json:"role"`
}

// SetRole godoc
// @Summary      Change a user's role
// @Description  Yalnız caller öz layihəsinin admin-idirsə icazə verilir.
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "User ID"
// @Param        request body setRoleRequest true "Role payload"
// @Success      200 {object} userResponse
// @Failure      403 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /users/{id}/role [post]
func (h *UserHandler) SetRole(w http.ResponseWriter, r *http.Request) {
	caller, ok := middleware.CallerFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req setRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Role != models.RoleUser && req.Role != models.RoleAdmin {
		writeError(w, http.StatusBadRequest, "role must be 'user' or 'admin'")
		return
	}

	id := chi.URLParam(r, "id")
	u, err := h.svc.SetRole(r.Context(), caller, id, req.Role)
	writeUserOrError(w, u, err)
}

type projectUserResponse struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

// ListByProject godoc
// @Summary      List a project's users
// @Description  Caller həmin layihənin admin-i olmalıdır.
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Project ID"
// @Success      200 {array} projectUserResponse
// @Failure      403 {object} map[string]string
// @Router       /projects/{id}/users [get]
func (h *UserHandler) ListByProject(w http.ResponseWriter, r *http.Request) {
	caller, ok := middleware.CallerFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	projectID := chi.URLParam(r, "id")
	users, err := h.svc.ListByProject(r.Context(), caller, projectID)
	if err != nil {
		if errors.Is(err, service.ErrForbidden) {
			writeError(w, http.StatusForbidden, "admin role within this project required")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to list users")
		return
	}

	response := make([]projectUserResponse, len(users))
	for i, u := range users {
		response[i] = projectUserResponse{ID: u.ID, Name: u.Name, Username: u.Username, Email: u.Email, Role: u.RoleName}
	}
	writeJSON(w, http.StatusOK, response)
}

func writeUserOrError(w http.ResponseWriter, u *models.User, err error) {
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrUserNotFound):
			writeError(w, http.StatusNotFound, "user not found")
		case errors.Is(err, service.ErrForbidden):
			writeError(w, http.StatusForbidden, "admin role within the target's own project required")
		default:
			writeError(w, http.StatusInternalServerError, "failed to process request")
		}
		return
	}

	projectID := ""
	if u.ProjectID != nil {
		projectID = *u.ProjectID
	}
	writeJSON(w, http.StatusOK, userResponse{
		ID: u.ID, Name: u.Name, Username: u.Username, Email: u.Email, ProjectID: projectID, Role: u.RoleName,
	})
}
