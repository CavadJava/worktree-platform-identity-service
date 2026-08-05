package handlers

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"shop-product-service/internal/middleware"
	_ "shop-product-service/internal/models" // referenced by swag annotations
	"shop-product-service/internal/service"
)

type FavoriteHandler struct {
	svc *service.FavoriteService
}

func NewFavoriteHandler(svc *service.FavoriteService) *FavoriteHandler {
	return &FavoriteHandler{svc: svc}
}

// Add godoc
// @Summary      Favorite a product
// @Tags         favorites
// @Security     BearerAuth
// @Param        id path string true "Product ID"
// @Success      204 "No Content"
// @Failure      404 {object} map[string]string
// @Router       /products/{id}/favorite [post]
func (h *FavoriteHandler) Add(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	productID := chi.URLParam(r, "id")
	if err := h.svc.Add(r.Context(), identity.UserID, productID); err != nil {
		if errors.Is(err, service.ErrProductNotFound) {
			writeError(w, http.StatusNotFound, "product not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to favorite product")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Remove godoc
// @Summary      Unfavorite a product
// @Tags         favorites
// @Security     BearerAuth
// @Param        id path string true "Product ID"
// @Success      204 "No Content"
// @Router       /products/{id}/favorite [delete]
func (h *FavoriteHandler) Remove(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	productID := chi.URLParam(r, "id")
	if err := h.svc.Remove(r.Context(), identity.UserID, productID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to unfavorite product")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// List godoc
// @Summary      List my favorite products
// @Tags         favorites
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} models.Product
// @Router       /favorites [get]
func (h *FavoriteHandler) List(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	products, err := h.svc.List(r.Context(), identity.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list favorites")
		return
	}

	writeJSON(w, http.StatusOK, products)
}
