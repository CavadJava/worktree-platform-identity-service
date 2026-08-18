package handlers

import (
	"net/http"

	"platform-identity-service/internal/service"
)

type RoleHandler struct {
	svc *service.RoleService
}

func NewRoleHandler(svc *service.RoleService) *RoleHandler {
	return &RoleHandler{svc: svc}
}

type roleResponse struct {
	ID   int16  `json:"id"`
	Name string `json:"name"`
}

// List godoc
// @Summary      List all roles
// @Description  Sabit 2 rolu qaytarır (user, admin).
// @Tags         roles
// @Produce      json
// @Success      200 {array} roleResponse
// @Router       /roles [get]
func (h *RoleHandler) List(w http.ResponseWriter, r *http.Request) {
	roles, err := h.svc.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list roles")
		return
	}

	response := make([]roleResponse, len(roles))
	for i, role := range roles {
		response[i] = roleResponse{ID: role.ID, Name: role.Name}
	}
	writeJSON(w, http.StatusOK, response)
}
