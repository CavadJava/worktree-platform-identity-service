package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"user-service/internal/middleware"
	_ "user-service/internal/models" // referenced by swag annotations
	"user-service/internal/service"
)

type ProfileHandler struct {
	svc *service.ProfileService
}

func NewProfileHandler(svc *service.ProfileService) *ProfileHandler {
	return &ProfileHandler{svc: svc}
}

// Get godoc
// @Summary      Get current user's profile
// @Description  JWT token-ə uyğun istifadəçinin profilini qaytarır (registration-service-in verdiyi token).
// @Tags         profile
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} models.User
// @Failure      401 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /profile [get]
func (h *ProfileHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := h.svc.GetProfile(r.Context(), userID)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load profile")
		return
	}

	writeJSON(w, http.StatusOK, user)
}

type updateProfileRequest struct {
	FullName string `json:"full_name"`
	Phone    string `json:"phone"`
}

// Update godoc
// @Summary      Update current user's profile
// @Description  full_name və phone sahələrini yeniləyir (email dəyişmir, registration-service tərəfindən idarə olunur).
// @Tags         profile
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body updateProfileRequest true "Profile update payload"
// @Success      200 {object} models.User
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /profile [put]
func (h *ProfileHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req updateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.FullName = strings.TrimSpace(req.FullName)
	if req.FullName == "" {
		writeError(w, http.StatusBadRequest, "full_name is required")
		return
	}

	user, err := h.svc.UpdateProfile(r.Context(), userID, req.FullName, req.Phone)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update profile")
		return
	}

	writeJSON(w, http.StatusOK, user)
}
