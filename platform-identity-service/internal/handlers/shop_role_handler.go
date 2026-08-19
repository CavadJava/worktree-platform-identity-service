package handlers

import (
	"net/http"

	"platform-identity-service/internal/service"
)

type ShopRoleHandler struct {
	svc *service.ShopRoleService
}

func NewShopRoleHandler(svc *service.ShopRoleService) *ShopRoleHandler {
	return &ShopRoleHandler{svc: svc}
}

type shopRoleResponse struct {
	ID   int16  `json:"id"`
	Name string `json:"name"`
}

// List godoc
// @Summary      List all shop roles
// @Tags         shop-roles
// @Produce      json
// @Success      200 {array} shopRoleResponse
// @Router       /shop-roles [get]
func (h *ShopRoleHandler) List(w http.ResponseWriter, r *http.Request) {
	roles, err := h.svc.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list shop roles")
		return
	}

	response := make([]shopRoleResponse, len(roles))
	for i, role := range roles {
		response[i] = shopRoleResponse{ID: role.ID, Name: role.Name}
	}
	writeJSON(w, http.StatusOK, response)
}
