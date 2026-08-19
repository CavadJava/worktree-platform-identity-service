package handlers

import (
	"net/http"

	"platform-identity-service/internal/service"
)

type SystemRoleHandler struct {
	svc *service.SystemRoleService
}

func NewSystemRoleHandler(svc *service.SystemRoleService) *SystemRoleHandler {
	return &SystemRoleHandler{svc: svc}
}

type systemRoleResponse struct {
	ID   int16  `json:"id"`
	Name string `json:"name"`
}

// List godoc
// @Summary      List all system roles
// @Tags         system-roles
// @Produce      json
// @Success      200 {array} systemRoleResponse
// @Router       /system-roles [get]
func (h *SystemRoleHandler) List(w http.ResponseWriter, r *http.Request) {
	roles, err := h.svc.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list system roles")
		return
	}

	response := make([]systemRoleResponse, len(roles))
	for i, role := range roles {
		response[i] = systemRoleResponse{ID: role.ID, Name: role.Name}
	}
	writeJSON(w, http.StatusOK, response)
}
