package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	_ "shop-category-service/internal/models" // referenced by swag annotations
	"shop-category-service/internal/service"
)

type CategoryHandler struct {
	svc *service.CategoryService
}

func NewCategoryHandler(svc *service.CategoryService) *CategoryHandler {
	return &CategoryHandler{svc: svc}
}

type categoryRequest struct {
	ID        string `json:"id,omitempty"` // boş isə addan slug yaradılır
	Name      string `json:"name"`
	Icon      string `json:"icon,omitempty"`
	SortOrder int    `json:"sort_order"`
}

type subcategoryRequest struct {
	ID        string `json:"id,omitempty"`
	Name      string `json:"name"`
	Image     string `json:"image,omitempty"`
	SortOrder int    `json:"sort_order"`
}

// ListCategories godoc
// @Summary      List all categories
// @Tags         categories
// @Produce      json
// @Success      200 {array} models.Category
// @Router       /categories [get]
func (h *CategoryHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := h.svc.ListCategories(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list categories")
		return
	}
	writeJSON(w, http.StatusOK, categories)
}

// GetCategory godoc
// @Summary      Get a category
// @Tags         categories
// @Produce      json
// @Param        id path string true "Category ID (slug)"
// @Success      200 {object} models.Category
// @Failure      404 {object} map[string]string
// @Router       /categories/{id} [get]
func (h *CategoryHandler) GetCategory(w http.ResponseWriter, r *http.Request) {
	c, err := h.svc.GetCategory(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeCategoryError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

// CreateCategory godoc
// @Summary      Create a category (administrator only)
// @Tags         categories
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body categoryRequest true "Category payload"
// @Success      201 {object} models.Category
// @Failure      400 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Failure      409 {object} map[string]string
// @Router       /categories [post]
func (h *CategoryHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var req categoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	c, err := h.svc.CreateCategory(r.Context(), req.ID, req.Name, req.Icon, req.SortOrder)
	if err != nil {
		writeCategoryError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

// UpdateCategory godoc
// @Summary      Update a category (administrator only)
// @Tags         categories
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Category ID (slug)"
// @Param        request body categoryRequest true "Category payload"
// @Success      200 {object} models.Category
// @Failure      400 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /categories/{id} [put]
func (h *CategoryHandler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	var req categoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	c, err := h.svc.UpdateCategory(r.Context(), chi.URLParam(r, "id"), req.Name, req.Icon, req.SortOrder)
	if err != nil {
		writeCategoryError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

// DeleteCategory godoc
// @Summary      Delete a category and its subcategories (administrator only)
// @Tags         categories
// @Security     BearerAuth
// @Param        id path string true "Category ID (slug)"
// @Success      204 "No Content"
// @Failure      403 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /categories/{id} [delete]
func (h *CategoryHandler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteCategory(r.Context(), chi.URLParam(r, "id")); err != nil {
		writeCategoryError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListSubcategories godoc
// @Summary      List a category's subcategories
// @Tags         categories
// @Produce      json
// @Param        id path string true "Category ID (slug)"
// @Success      200 {array} models.Subcategory
// @Failure      404 {object} map[string]string
// @Router       /categories/{id}/subcategories [get]
func (h *CategoryHandler) ListSubcategories(w http.ResponseWriter, r *http.Request) {
	subs, err := h.svc.ListSubcategories(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeCategoryError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, subs)
}

// ListAllSubcategories godoc
// @Summary      List all subcategories across categories
// @Tags         categories
// @Produce      json
// @Success      200 {array} models.Subcategory
// @Router       /subcategories [get]
func (h *CategoryHandler) ListAllSubcategories(w http.ResponseWriter, r *http.Request) {
	subs, err := h.svc.ListSubcategories(r.Context(), "")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list subcategories")
		return
	}
	writeJSON(w, http.StatusOK, subs)
}

// CreateSubcategory godoc
// @Summary      Add a subcategory to a category (administrator only)
// @Tags         categories
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Category ID (slug)"
// @Param        request body subcategoryRequest true "Subcategory payload"
// @Success      201 {object} models.Subcategory
// @Failure      400 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Failure      409 {object} map[string]string
// @Router       /categories/{id}/subcategories [post]
func (h *CategoryHandler) CreateSubcategory(w http.ResponseWriter, r *http.Request) {
	var req subcategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	sub, err := h.svc.CreateSubcategory(r.Context(), chi.URLParam(r, "id"), req.ID, req.Name, req.Image, req.SortOrder)
	if err != nil {
		writeCategoryError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, sub)
}

// UpdateSubcategory godoc
// @Summary      Update a subcategory (administrator only)
// @Tags         categories
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Subcategory ID (slug)"
// @Param        request body subcategoryRequest true "Subcategory payload"
// @Success      200 {object} models.Subcategory
// @Failure      400 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /subcategories/{id} [put]
func (h *CategoryHandler) UpdateSubcategory(w http.ResponseWriter, r *http.Request) {
	var req subcategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	sub, err := h.svc.UpdateSubcategory(r.Context(), chi.URLParam(r, "id"), req.Name, req.Image, req.SortOrder)
	if err != nil {
		writeCategoryError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sub)
}

// DeleteSubcategory godoc
// @Summary      Delete a subcategory (administrator only)
// @Tags         categories
// @Security     BearerAuth
// @Param        id path string true "Subcategory ID (slug)"
// @Success      204 "No Content"
// @Failure      403 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /subcategories/{id} [delete]
func (h *CategoryHandler) DeleteSubcategory(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteSubcategory(r.Context(), chi.URLParam(r, "id")); err != nil {
		writeCategoryError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeCategoryError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrCategoryNotFound):
		writeError(w, http.StatusNotFound, "category not found")
	case errors.Is(err, service.ErrSubcategoryNotFound):
		writeError(w, http.StatusNotFound, "subcategory not found")
	case errors.Is(err, service.ErrIDExists):
		writeError(w, http.StatusConflict, "id already exists")
	default:
		writeError(w, http.StatusInternalServerError, "failed to process request")
	}
}
