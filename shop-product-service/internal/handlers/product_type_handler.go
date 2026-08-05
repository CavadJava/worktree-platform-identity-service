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

type ProductTypeHandler struct {
	svc *service.ProductTypeService
}

func NewProductTypeHandler(svc *service.ProductTypeService) *ProductTypeHandler {
	return &ProductTypeHandler{svc: svc}
}

type productTypeRequest struct {
	ShopID string `json:"shop_id"`
	Name   string `json:"name"`
}

type productSubtypeRequest struct {
	Name string `json:"name"`
}

// CreateType godoc
// @Summary      Create a product type (növ) for a shop
// @Description  Məsələn "Ölçü". add-product(2)+ səviyyəli mağaza əməkdaşı və ya administrator əlavə edə bilər.
// @Tags         product-types
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body productTypeRequest true "Product type payload"
// @Success      201 {object} models.ProductType
// @Failure      400 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Router       /product-types [post]
func (h *ProductTypeHandler) CreateType(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req productTypeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ShopID == "" || req.Name == "" {
		writeError(w, http.StatusBadRequest, "shop_id and name are required")
		return
	}

	t, err := h.svc.CreateType(r.Context(), toIdentity(identity), req.ShopID, req.Name)
	if err != nil {
		writeTypeError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, t)
}

// ListTypes godoc
// @Summary      List a shop's product types
// @Tags         product-types
// @Produce      json
// @Param        shop_id query string true "Shop ID"
// @Success      200 {array} models.ProductType
// @Router       /product-types [get]
func (h *ProductTypeHandler) ListTypes(w http.ResponseWriter, r *http.Request) {
	shopID := r.URL.Query().Get("shop_id")
	if shopID == "" {
		writeError(w, http.StatusBadRequest, "shop_id is required")
		return
	}
	types, err := h.svc.ListTypes(r.Context(), shopID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list product types")
		return
	}
	writeJSON(w, http.StatusOK, types)
}

// GetType godoc
// @Summary      Get a product type
// @Tags         product-types
// @Produce      json
// @Param        id path string true "Product type ID"
// @Success      200 {object} models.ProductType
// @Failure      404 {object} map[string]string
// @Router       /product-types/{id} [get]
func (h *ProductTypeHandler) GetType(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	t, err := h.svc.GetType(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrProductTypeNotFound) {
			writeError(w, http.StatusNotFound, "product type not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load product type")
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// UpdateType godoc
// @Summary      Update a product type
// @Description  add-product(2)+ səviyyəli mağaza əməkdaşı və ya administrator.
// @Tags         product-types
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Product type ID"
// @Param        request body productSubtypeRequest true "Name payload"
// @Success      200 {object} models.ProductType
// @Failure      400 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /product-types/{id} [put]
func (h *ProductTypeHandler) UpdateType(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")

	var req productSubtypeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	t, err := h.svc.UpdateType(r.Context(), toIdentity(identity), id, req.Name)
	if err != nil {
		writeTypeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, t)
}

// DeleteType godoc
// @Summary      Delete a product type
// @Description  review(3)+ səviyyəli mağaza əməkdaşı və ya administrator.
// @Tags         product-types
// @Security     BearerAuth
// @Param        id path string true "Product type ID"
// @Success      204 "No Content"
// @Failure      403 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /product-types/{id} [delete]
func (h *ProductTypeHandler) DeleteType(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	if err := h.svc.DeleteType(r.Context(), toIdentity(identity), id); err != nil {
		writeTypeError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// CreateSubtype godoc
// @Summary      Add a subtype (alt növ) to a product type
// @Description  Məsələn "Ölçü" növünə "En", "Uzunluq" alt növləri. add-product(2)+ səviyyəli mağaza əməkdaşı və ya administrator.
// @Tags         product-types
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        type_id path string true "Product type ID"
// @Param        request body productSubtypeRequest true "Subtype payload"
// @Success      201 {object} models.ProductSubtype
// @Failure      400 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /product-types/{type_id}/subtypes [post]
func (h *ProductTypeHandler) CreateSubtype(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	typeID := chi.URLParam(r, "type_id")

	var req productSubtypeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	st, err := h.svc.CreateSubtype(r.Context(), toIdentity(identity), typeID, req.Name)
	if err != nil {
		writeTypeError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, st)
}

// ListSubtypes godoc
// @Summary      List a product type's subtypes
// @Tags         product-types
// @Produce      json
// @Param        type_id path string true "Product type ID"
// @Success      200 {array} models.ProductSubtype
// @Router       /product-types/{type_id}/subtypes [get]
func (h *ProductTypeHandler) ListSubtypes(w http.ResponseWriter, r *http.Request) {
	typeID := chi.URLParam(r, "type_id")
	subtypes, err := h.svc.ListSubtypes(r.Context(), typeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list product subtypes")
		return
	}
	writeJSON(w, http.StatusOK, subtypes)
}

// UpdateSubtype godoc
// @Summary      Update a product subtype
// @Description  add-product(2)+ səviyyəli mağaza əməkdaşı və ya administrator.
// @Tags         product-types
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Product subtype ID"
// @Param        request body productSubtypeRequest true "Name payload"
// @Success      200 {object} models.ProductSubtype
// @Failure      400 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /product-subtypes/{id} [put]
func (h *ProductTypeHandler) UpdateSubtype(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")

	var req productSubtypeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	st, err := h.svc.UpdateSubtype(r.Context(), toIdentity(identity), id, req.Name)
	if err != nil {
		writeTypeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, st)
}

// DeleteSubtype godoc
// @Summary      Delete a product subtype
// @Description  review(3)+ səviyyəli mağaza əməkdaşı və ya administrator.
// @Tags         product-types
// @Security     BearerAuth
// @Param        id path string true "Product subtype ID"
// @Success      204 "No Content"
// @Failure      403 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /product-subtypes/{id} [delete]
func (h *ProductTypeHandler) DeleteSubtype(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	if err := h.svc.DeleteSubtype(r.Context(), toIdentity(identity), id); err != nil {
		writeTypeError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeTypeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrProductTypeNotFound):
		writeError(w, http.StatusNotFound, "product type not found")
	case errors.Is(err, service.ErrProductSubtypeNotFound):
		writeError(w, http.StatusNotFound, "product subtype not found")
	case errors.Is(err, service.ErrForbidden):
		writeError(w, http.StatusForbidden, "insufficient permission: must belong to this shop with at least the required hierarchical level, or be an administrator")
	default:
		writeError(w, http.StatusInternalServerError, "failed to process request")
	}
}
