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

type ShopMembershipHandler struct {
	svc *service.ShopMembershipService
}

func NewShopMembershipHandler(svc *service.ShopMembershipService) *ShopMembershipHandler {
	return &ShopMembershipHandler{svc: svc}
}

type addMemberRequest struct {
	UserID   string `json:"user_id"`
	ShopRole string `json:"shop_role"`
}

type memberResponse struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	ShopID    string `json:"shop_id"`
	ShopName  string `json:"shop_name"`
	ShopRole  string `json:"shop_role"`
	CreatedAt string `json:"created_at"`
}

func toMemberResponse(m *models.ShopMembership) memberResponse {
	return memberResponse{
		ID: m.ID, UserID: m.UserID, ShopID: m.ShopID, ShopName: m.ShopName,
		ShopRole: m.ShopRoleName, CreatedAt: m.CreatedAt.Format(timeFormat),
	}
}

// AddMember godoc
// @Summary      Add an existing user to a shop
// @Description  Caller must be superadmin or that shop's own shop-admin.
// @Tags         shops
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Shop ID"
// @Param        request body addMemberRequest true "Member payload"
// @Success      201 {object} memberResponse
// @Failure      400 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Router       /shops/{id}/members [post]
func (h *ShopMembershipHandler) AddMember(w http.ResponseWriter, r *http.Request) {
	caller, ok := middleware.CallerFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req addMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.UserID == "" {
		writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	shopID := chi.URLParam(r, "id")
	m, err := h.svc.AddMember(r.Context(), caller, shopID, req.UserID, req.ShopRole)
	if err != nil {
		writeMembershipError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toMemberResponse(m))
}

// ListMembers godoc
// @Summary      List a shop's members
// @Tags         shops
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Shop ID"
// @Success      200 {array} memberResponse
// @Failure      403 {object} map[string]string
// @Router       /shops/{id}/members [get]
func (h *ShopMembershipHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	caller, ok := middleware.CallerFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	shopID := chi.URLParam(r, "id")
	members, err := h.svc.ListMembers(r.Context(), caller, shopID)
	if err != nil {
		writeMembershipError(w, err)
		return
	}

	response := make([]memberResponse, len(members))
	for i := range members {
		response[i] = toMemberResponse(&members[i])
	}
	writeJSON(w, http.StatusOK, response)
}

type setMemberRoleRequest struct {
	ShopRole string `json:"shop_role"`
}

// SetMemberRole godoc
// @Summary      Change a member's shop role
// @Tags         shops
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Shop ID"
// @Param        userId path string true "User ID"
// @Param        request body setMemberRoleRequest true "Role payload"
// @Success      200 {object} memberResponse
// @Failure      403 {object} map[string]string
// @Router       /shops/{id}/members/{userId}/role [post]
func (h *ShopMembershipHandler) SetMemberRole(w http.ResponseWriter, r *http.Request) {
	caller, ok := middleware.CallerFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req setMemberRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	shopID := chi.URLParam(r, "id")
	userID := chi.URLParam(r, "userId")
	m, err := h.svc.SetMemberRole(r.Context(), caller, shopID, userID, req.ShopRole)
	if err != nil {
		writeMembershipError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toMemberResponse(m))
}

func writeMembershipError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrForbidden):
		writeError(w, http.StatusForbidden, "superadmin or this shop's own shop-admin required")
	case errors.Is(err, service.ErrInvalidShopRole):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrMembershipExists):
		writeError(w, http.StatusConflict, "user is already a member of this shop")
	case errors.Is(err, service.ErrMembershipNotFound):
		writeError(w, http.StatusNotFound, "membership not found")
	case errors.Is(err, repository.ErrShopNotFound):
		writeError(w, http.StatusNotFound, "shop not found")
	case errors.Is(err, repository.ErrUserNotFound):
		writeError(w, http.StatusNotFound, "user not found")
	default:
		writeError(w, http.StatusInternalServerError, "failed to process request")
	}
}
