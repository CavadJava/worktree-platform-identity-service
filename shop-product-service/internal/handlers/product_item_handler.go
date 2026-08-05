package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"shop-product-service/internal/client"
	"shop-product-service/internal/middleware"
	_ "shop-product-service/internal/models" // referenced by swag annotations
	"shop-product-service/internal/service"
)

type ProductItemHandler struct {
	svc *service.ProductItemService
}

func NewProductItemHandler(svc *service.ProductItemService) *ProductItemHandler {
	return &ProductItemHandler{svc: svc}
}

type productItemRequest struct {
	Name          string   `json:"name"`
	Price         float64  `json:"price"`
	Stock         int      `json:"stock"`
	IsDiscounted  bool     `json:"is_discounted"`
	DiscountPrice *float64 `json:"discount_price,omitempty"`
}

// Create godoc
// @Summary      Add a variant/item to a product
// @Description  Məsələn "Bayraq" məhsuluna "30x60 1 qat", "100x100 2 qat" kimi hər birinin öz qiyməti/stoku/endirimi olan itemlər əlavə edilir. add-product(2)+ səviyyəli mağaza əməkdaşı və ya administrator.
// @Tags         product-items
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        product_id path string true "Product ID"
// @Param        request body productItemRequest true "Item payload"
// @Success      201 {object} models.ProductItem
// @Failure      400 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /products/{product_id}/items [post]
func (h *ProductItemHandler) Create(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	productID := chi.URLParam(r, "product_id")

	var req productItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	item, err := h.svc.Create(r.Context(), toIdentity(identity), productID, req.Name, req.Price, req.Stock, req.IsDiscounted, req.DiscountPrice)
	if err != nil {
		writeItemError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, item)
}

// List godoc
// @Summary      List a product's items/variants
// @Tags         product-items
// @Produce      json
// @Param        product_id path string true "Product ID"
// @Success      200 {array} models.ProductItem
// @Router       /products/{product_id}/items [get]
func (h *ProductItemHandler) List(w http.ResponseWriter, r *http.Request) {
	productID := chi.URLParam(r, "product_id")
	items, err := h.svc.ListByProduct(r.Context(), productID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list product items")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// Get godoc
// @Summary      Get a product item
// @Tags         product-items
// @Produce      json
// @Param        id path string true "Product item ID"
// @Success      200 {object} models.ProductItem
// @Failure      404 {object} map[string]string
// @Router       /product-items/{id} [get]
func (h *ProductItemHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	item, err := h.svc.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrProductItemNotFound) {
			writeError(w, http.StatusNotFound, "product item not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load product item")
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// Update godoc
// @Summary      Update a product item
// @Description  add-product(2)+ səviyyəli mağaza əməkdaşı və ya administrator.
// @Tags         product-items
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Product item ID"
// @Param        request body productItemRequest true "Item payload"
// @Success      200 {object} models.ProductItem
// @Failure      400 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /product-items/{id} [put]
func (h *ProductItemHandler) Update(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")

	var req productItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	item, err := h.svc.Update(r.Context(), toIdentity(identity), id, req.Name, req.Price, req.Stock, req.IsDiscounted, req.DiscountPrice)
	if err != nil {
		writeItemError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, item)
}

// Delete godoc
// @Summary      Delete a product item
// @Description  review(3)+ səviyyəli mağaza əməkdaşı və ya administrator.
// @Tags         product-items
// @Security     BearerAuth
// @Param        id path string true "Product item ID"
// @Success      204 "No Content"
// @Failure      403 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /product-items/{id} [delete]
func (h *ProductItemHandler) Delete(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	if err := h.svc.Delete(r.Context(), toIdentity(identity), id); err != nil {
		writeItemError(w, err)
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

func writeItemError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrProductNotFound):
		writeError(w, http.StatusNotFound, "product not found")
	case errors.Is(err, service.ErrProductItemNotFound):
		writeError(w, http.StatusNotFound, "product item not found")
	case errors.Is(err, service.ErrForbidden):
		writeError(w, http.StatusForbidden, "insufficient permission: must belong to this shop with at least the required hierarchical level, or be an administrator")
	case errors.Is(err, service.ErrDiscountPriceMissing), errors.Is(err, service.ErrDiscountPriceInvalid):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "failed to process request")
	}
}
