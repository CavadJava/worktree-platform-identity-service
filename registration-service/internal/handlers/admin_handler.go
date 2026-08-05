package handlers

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	_ "registration-service/internal/models" // referenced by swag annotations
	"registration-service/internal/service"
)

type AdminHandler struct {
	svc *service.UserService
}

func NewAdminHandler(svc *service.UserService) *AdminHandler {
	return &AdminHandler{svc: svc}
}

// ListUsers godoc
// @Summary      List users (administrator only)
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Param        email query string false "Filter by email (partial match)"
// @Success      200 {array} models.User
// @Failure      403 {object} map[string]string
// @Router       /users [get]
func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	emailFilter := r.URL.Query().Get("email")
	users, err := h.svc.ListUsers(r.Context(), emailFilter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list users")
		return
	}
	writeJSON(w, http.StatusOK, users)
}

// GetUser godoc
// @Summary      Get a user by ID (administrator only)
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "User ID"
// @Success      200 {object} models.User
// @Failure      403 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /users/{id} [get]
func (h *AdminHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	user, err := h.svc.GetUser(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load user")
		return
	}
	writeJSON(w, http.StatusOK, user)
}
