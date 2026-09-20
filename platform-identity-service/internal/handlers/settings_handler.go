package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"platform-identity-service/internal/middleware"
	"platform-identity-service/internal/service"
)

type SettingsHandler struct {
	svc *service.SettingsService
}

func NewSettingsHandler(svc *service.SettingsService) *SettingsHandler {
	return &SettingsHandler{svc: svc}
}

type jwtTTLResponse struct {
	JWTTTLMinutes int `json:"jwt_ttl_minutes"`
}

// GetJWTTTL godoc
// @Summary      Get the current login token lifetime, in minutes
// @Description  Superadmin only.
// @Tags         settings
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} jwtTTLResponse
// @Failure      403 {object} map[string]string
// @Router       /settings/jwt-ttl [get]
func (h *SettingsHandler) GetJWTTTL(w http.ResponseWriter, r *http.Request) {
	caller, ok := middleware.CallerFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	minutes, err := h.svc.GetJWTTTLMinutes(r.Context(), caller)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrForbidden):
			writeError(w, http.StatusForbidden, "superadmin role required")
		default:
			writeError(w, http.StatusInternalServerError, "failed to read setting")
		}
		return
	}
	writeJSON(w, http.StatusOK, jwtTTLResponse{JWTTTLMinutes: minutes})
}

type setJWTTTLRequest struct {
	JWTTTLMinutes int `json:"jwt_ttl_minutes"`
}

// SetJWTTTL godoc
// @Summary      Set the login token lifetime, in minutes
// @Description  Superadmin only. Takes effect for every new login immediately, no restart required.
// @Tags         settings
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body setJWTTTLRequest true "New TTL payload"
// @Success      200 {object} jwtTTLResponse
// @Failure      400 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Router       /settings/jwt-ttl [post]
func (h *SettingsHandler) SetJWTTTL(w http.ResponseWriter, r *http.Request) {
	caller, ok := middleware.CallerFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req setJWTTTLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.svc.SetJWTTTLMinutes(r.Context(), caller, req.JWTTTLMinutes); err != nil {
		switch {
		case errors.Is(err, service.ErrForbidden):
			writeError(w, http.StatusForbidden, "superadmin role required")
		case errors.Is(err, service.ErrInvalidTTL):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "failed to update setting")
		}
		return
	}
	writeJSON(w, http.StatusOK, jwtTTLResponse{JWTTTLMinutes: req.JWTTTLMinutes})
}
