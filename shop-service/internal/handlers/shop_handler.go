package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"shop-service/internal/client"
	"shop-service/internal/middleware"
	_ "shop-service/internal/models" // referenced by swag annotations
	"shop-service/internal/service"
)

type ShopHandler struct {
	svc *service.ShopService
}

func NewShopHandler(svc *service.ShopService) *ShopHandler {
	return &ShopHandler{svc: svc}
}

type shopRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Create godoc
// @Summary      Create a shop directly (administrator only)
// @Description  Adi istifadəçilər mağazanı bu endpoint ilə deyil, /shop-applications müraciət+təsdiq axını ilə açır. Bu endpoint yalnız administratorun birbaşa yaratması üçündür.
// @Tags         shops
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body shopRequest true "Shop payload"
// @Success      201 {object} models.Shop
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Router       /shops [post]
func (h *ShopHandler) Create(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req shopRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	shop, err := h.svc.Create(r.Context(), toIdentity(identity), req.Name, req.Description)
	if err != nil {
		if errors.Is(err, service.ErrForbidden) {
			writeError(w, http.StatusForbidden, "administrator role required")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create shop")
		return
	}

	writeJSON(w, http.StatusCreated, shop)
}

// List godoc
// @Summary      List shops
// @Tags         shops
// @Produce      json
// @Param        owner_id query string false "Filter by owner"
// @Success      200 {array} models.Shop
// @Router       /shops [get]
func (h *ShopHandler) List(w http.ResponseWriter, r *http.Request) {
	ownerID := r.URL.Query().Get("owner_id")
	shops, err := h.svc.List(r.Context(), ownerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list shops")
		return
	}
	writeJSON(w, http.StatusOK, shops)
}

// Get godoc
// @Summary      Get a shop
// @Tags         shops
// @Produce      json
// @Param        id path string true "Shop ID"
// @Success      200 {object} models.Shop
// @Failure      404 {object} map[string]string
// @Router       /shops/{id} [get]
func (h *ShopHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	shop, err := h.svc.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrShopNotFound) {
			writeError(w, http.StatusNotFound, "shop not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load shop")
		return
	}
	writeJSON(w, http.StatusOK, shop)
}

// Update godoc
// @Summary      Update a shop
// @Description  review(3) və admin(4) səviyyəli öz mağaza əməkdaşları və ya administrator.
// @Tags         shops
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Shop ID"
// @Param        request body shopRequest true "Shop payload"
// @Success      200 {object} models.Shop
// @Failure      400 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /shops/{id} [put]
func (h *ShopHandler) Update(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")

	var req shopRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	shop, err := h.svc.Update(r.Context(), toIdentity(identity), id, req.Name, req.Description)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrShopNotFound):
			writeError(w, http.StatusNotFound, "shop not found")
		case errors.Is(err, service.ErrForbidden):
			writeError(w, http.StatusForbidden, "review(3)+ level in this shop, or administrator, required")
		default:
			writeError(w, http.StatusInternalServerError, "failed to update shop")
		}
		return
	}

	writeJSON(w, http.StatusOK, shop)
}

// Delete godoc
// @Summary      Delete a shop
// @Description  Yalnız admin(4) səviyyəli mağaza əməkdaşı və ya administrator.
// @Tags         shops
// @Security     BearerAuth
// @Param        id path string true "Shop ID"
// @Success      204 "No Content"
// @Failure      403 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /shops/{id} [delete]
func (h *ShopHandler) Delete(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")

	err := h.svc.Delete(r.Context(), toIdentity(identity), id)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrShopNotFound):
			writeError(w, http.StatusNotFound, "shop not found")
		case errors.Is(err, service.ErrForbidden):
			writeError(w, http.StatusForbidden, "admin(4) level in this shop, or administrator, required")
		default:
			writeError(w, http.StatusInternalServerError, "failed to delete shop")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func toIdentity(identity *client.Identity) service.Identity {
	return service.Identity{
		UserID:        identity.UserID,
		Role:          identity.Role,
		ShopID:        identity.ShopID,
		ShopRoleLevel: identity.ShopRoleLevel,
	}
}
