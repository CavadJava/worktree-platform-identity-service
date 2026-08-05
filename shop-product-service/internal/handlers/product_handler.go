package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"shop-product-service/internal/middleware"
	_ "shop-product-service/internal/models" // referenced by swag annotations
	"shop-product-service/internal/service"
)

type ProductHandler struct {
	svc *service.ProductService
}

func NewProductHandler(svc *service.ProductService) *ProductHandler {
	return &ProductHandler{svc: svc}
}

type productRequest struct {
	ShopID      string  `json:"shop_id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Stock       int     `json:"stock"`
}

// Create godoc
// @Summary      Create a product
// @Description  add-product(2)+ səviyyəsi olan mağaza əməkdaşı və ya administrator əlavə edə bilər.
// @Tags         products
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body productRequest true "Product payload"
// @Success      201 {object} models.Product
// @Failure      400 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /products [post]
func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req productRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ShopID == "" || req.Name == "" {
		writeError(w, http.StatusBadRequest, "shop_id and name are required")
		return
	}

	product, err := h.svc.Create(r.Context(), service.Identity{
		UserID:        identity.UserID,
		Role:          identity.Role,
		ShopID:        identity.ShopID,
		ShopRoleLevel: identity.ShopRoleLevel,
	}, req.ShopID, req.Name, req.Description, req.Price, req.Stock)
	if err != nil {
		writeMutationError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, product)
}

// List godoc
// @Summary      List products
// @Tags         products
// @Produce      json
// @Param        shop_id query string false "Filter by shop"
// @Success      200 {array} models.Product
// @Router       /products [get]
func (h *ProductHandler) List(w http.ResponseWriter, r *http.Request) {
	shopID := r.URL.Query().Get("shop_id")
	products, err := h.svc.List(r.Context(), shopID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list products")
		return
	}
	writeJSON(w, http.StatusOK, products)
}

// Get godoc
// @Summary      Get a product
// @Tags         products
// @Produce      json
// @Param        id path string true "Product ID"
// @Success      200 {object} models.Product
// @Failure      404 {object} map[string]string
// @Router       /products/{id} [get]
func (h *ProductHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	product, err := h.svc.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrProductNotFound) {
			writeError(w, http.StatusNotFound, "product not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load product")
		return
	}
	writeJSON(w, http.StatusOK, product)
}

// Update godoc
// @Summary      Update a product
// @Description  add-product(2)+ səviyyəsi olan mağaza əməkdaşı və ya administrator.
// @Tags         products
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Product ID"
// @Param        request body productRequest true "Product payload"
// @Success      200 {object} models.Product
// @Failure      400 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /products/{id} [put]
func (h *ProductHandler) Update(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")

	var req productRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	product, err := h.svc.Update(r.Context(), service.Identity{
		UserID:        identity.UserID,
		Role:          identity.Role,
		ShopID:        identity.ShopID,
		ShopRoleLevel: identity.ShopRoleLevel,
	}, id, req.Name, req.Description, req.Price, req.Stock)
	if err != nil {
		writeMutationError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, product)
}

// Delete godoc
// @Summary      Delete a product
// @Description  review(3)+ səviyyəsi olan mağaza əməkdaşı və ya administrator.
// @Tags         products
// @Security     BearerAuth
// @Param        id path string true "Product ID"
// @Success      204 "No Content"
// @Failure      403 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /products/{id} [delete]
func (h *ProductHandler) Delete(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")

	err := h.svc.Delete(r.Context(), service.Identity{
		UserID:        identity.UserID,
		Role:          identity.Role,
		ShopID:        identity.ShopID,
		ShopRoleLevel: identity.ShopRoleLevel,
	}, id)
	if err != nil {
		writeMutationError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeMutationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrProductNotFound):
		writeError(w, http.StatusNotFound, "product not found")
	case errors.Is(err, service.ErrForbidden):
		writeError(w, http.StatusForbidden, "insufficient permission: must belong to this shop with at least the required hierarchical level, or be an administrator")
	default:
		writeError(w, http.StatusInternalServerError, "failed to process request")
	}
}
