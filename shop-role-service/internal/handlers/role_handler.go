package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"shop-role-service/internal/middleware"
	"shop-role-service/internal/service"
)

type RoleHandler struct {
	svc *service.RoleService
}

func NewRoleHandler(svc *service.RoleService) *RoleHandler {
	return &RoleHandler{svc: svc}
}

type assignRequest struct {
	UserID        string `json:"user_id"`
	ShopID        string `json:"shop_id"`
	ShopRoleLevel int    `json:"shop_role_level"`
}

type revokeRequest struct {
	UserID string `json:"user_id"`
}

type assignmentResponse struct {
	UserID        string  `json:"user_id"`
	ShopID        *string `json:"shop_id,omitempty"`
	ShopRoleLevel int     `json:"shop_role_level"`
}

// Assign godoc
// @Summary      Assign a user to a shop with a hierarchical level
// @Description  admin(4) > review(3) > add-product(2) > chat(1). Sistem administratoru istənilən mağazaya, mağazanın öz admin(4)-ü isə yalnız öz mağazasına əməkdaş təyin edə bilər. Bir istifadəçi eyni anda yalnız bir mağazaya aid ola bilər.
// @Tags         roles
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body assignRequest true "Assign payload"
// @Success      200 {object} assignmentResponse
// @Failure      400 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /roles/assign [post]
func (h *RoleHandler) Assign(w http.ResponseWriter, r *http.Request) {
	caller, ok := callerFromContext(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req assignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.UserID == "" {
		writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	assignment, err := h.svc.Assign(r.Context(), caller, req.UserID, req.ShopID, req.ShopRoleLevel)
	writeAssignment(w, assignment, err)
}

// Revoke godoc
// @Summary      Remove a user from their shop
// @Description  shop_id-ni sıfırlayır və şəviyyəni 0-a endirir. Sistem administratoru istənilən istifadəçini, mağazanın öz admin(4)-ü isə yalnız öz mağazasındakı əməkdaşı çıxara bilər.
// @Tags         roles
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body revokeRequest true "Revoke payload"
// @Success      200 {object} assignmentResponse
// @Failure      400 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /roles/revoke [post]
func (h *RoleHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	caller, ok := callerFromContext(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req revokeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.UserID == "" {
		writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	assignment, err := h.svc.Revoke(r.Context(), caller, req.UserID)
	writeAssignment(w, assignment, err)
}

// Get godoc
// @Summary      Get a user's shop assignment
// @Description  Sistem administratoru istənilən istifadəçini, mağazanın öz admin(4)-ü isə yalnız öz mağazasındakı əməkdaşları görə bilər.
// @Tags         roles
// @Produce      json
// @Security     BearerAuth
// @Param        user_id path string true "User ID"
// @Success      200 {object} assignmentResponse
// @Failure      403 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /roles/{user_id} [get]
func (h *RoleHandler) Get(w http.ResponseWriter, r *http.Request) {
	caller, ok := callerFromContext(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	userID := chi.URLParam(r, "user_id")

	assignment, err := h.svc.Get(r.Context(), caller, userID)
	writeAssignment(w, assignment, err)
}

type staffMemberResponse struct {
	UserID        string `json:"user_id"`
	Email         string `json:"email"`
	FullName      string `json:"full_name"`
	ShopRoleLevel int    `json:"shop_role_level"`
}

// ListStaff godoc
// @Summary      List a shop's staff
// @Description  Sistem administratoru istənilən mağazanın, mağazanın öz admin(4)-ü isə yalnız öz mağazasının əməkdaşlarını görə bilər.
// @Tags         roles
// @Produce      json
// @Security     BearerAuth
// @Param        shop_id path string true "Shop ID"
// @Success      200 {array} staffMemberResponse
// @Failure      403 {object} map[string]string
// @Router       /shops/{shop_id}/staff [get]
func (h *RoleHandler) ListStaff(w http.ResponseWriter, r *http.Request) {
	caller, ok := callerFromContext(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	shopID := chi.URLParam(r, "shop_id")
	staff, err := h.svc.ListStaff(r.Context(), caller, shopID)
	if err != nil {
		if errors.Is(err, service.ErrForbidden) {
			writeError(w, http.StatusForbidden, "administrator, or the shop's own admin(4) for this specific shop, required")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to list staff")
		return
	}

	response := make([]staffMemberResponse, len(staff))
	for i, m := range staff {
		response[i] = staffMemberResponse{UserID: m.UserID, Email: m.Email, FullName: m.FullName, ShopRoleLevel: m.ShopRoleLevel}
	}
	writeJSON(w, http.StatusOK, response)
}

func callerFromContext(r *http.Request) (service.Caller, bool) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		return service.Caller{}, false
	}
	return service.Caller{
		Role:          identity.Role,
		ShopID:        identity.ShopID,
		ShopRoleLevel: identity.ShopRoleLevel,
	}, true
}

func writeAssignment(w http.ResponseWriter, assignment *service.Assignment, err error) {
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			writeError(w, http.StatusNotFound, "user not found")
		case errors.Is(err, service.ErrInvalidShopLevel), errors.Is(err, service.ErrShopIDRequired):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, service.ErrForbidden):
			writeError(w, http.StatusForbidden, "administrator, or the shop's own admin(4) for this specific shop, required")
		default:
			writeError(w, http.StatusInternalServerError, "failed to process request")
		}
		return
	}

	writeJSON(w, http.StatusOK, assignmentResponse{
		UserID:        assignment.UserID,
		ShopID:        assignment.ShopID,
		ShopRoleLevel: assignment.ShopRoleLevel,
	})
}
