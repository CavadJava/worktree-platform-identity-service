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

type ProductHandler struct {
	svc *service.ProductService
}

func NewProductHandler(svc *service.ProductService) *ProductHandler {
	return &ProductHandler{svc: svc}
}

type createProductRequest struct {
	Name string `json:"name"`
}

type productResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

// Create godoc
// @Summary      Create a product
// @Description  Superadmin only.
// @Tags         products
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body createProductRequest true "Product payload"
// @Success      201 {object} productResponse
// @Failure      403 {object} map[string]string
// @Router       /products [post]
func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) {
	caller, ok := middleware.CallerFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if caller.SystemRole != models.SystemRoleSuperadmin {
		writeError(w, http.StatusForbidden, "superadmin role required")
		return
	}

	var req createProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	p, err := h.svc.Create(r.Context(), req.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create product")
		return
	}
	writeJSON(w, http.StatusCreated, productResponse{ID: p.ID, Name: p.Name, CreatedAt: p.CreatedAt.Format(timeFormat)})
}

// List godoc
// @Summary      List all products
// @Tags         products
// @Produce      json
// @Success      200 {array} productResponse
// @Router       /products [get]
func (h *ProductHandler) List(w http.ResponseWriter, r *http.Request) {
	products, err := h.svc.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list products")
		return
	}

	response := make([]productResponse, len(products))
	for i, p := range products {
		response[i] = productResponse{ID: p.ID, Name: p.Name, CreatedAt: p.CreatedAt.Format(timeFormat)}
	}
	writeJSON(w, http.StatusOK, response)
}

type accessResponse struct {
	Access string `json:"access"`
}

// CheckAccess godoc
// @Summary      Check the caller's access level to a product
// @Description  Returns "full" if subscribed, "demo" otherwise.
// @Tags         products
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Product ID"
// @Success      200 {object} accessResponse
// @Failure      404 {object} map[string]string
// @Router       /products/{id}/access [get]
func (h *ProductHandler) CheckAccess(w http.ResponseWriter, r *http.Request) {
	caller, ok := middleware.CallerFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	productID := chi.URLParam(r, "id")
	access, err := h.svc.CheckAccess(r.Context(), caller.UserID, productID)
	if err != nil {
		if errors.Is(err, repository.ErrProductNotFound) {
			writeError(w, http.StatusNotFound, "product not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to check access")
		return
	}
	writeJSON(w, http.StatusOK, accessResponse{Access: access})
}

type setSubscriptionRequest struct {
	Subscripted bool `json:"subscripted"`
	Renewed     bool `json:"renewed"`
}

type subscriptionResponse struct {
	UserID      string `json:"user_id"`
	ProductID   string `json:"product_id"`
	Subscripted bool   `json:"subscripted"`
	Renewed     bool   `json:"renewed"`
}

// SetSubscription godoc
// @Summary      Manually set a user's subscription to a product
// @Description  Superadmin only. No payment gateway integration.
// @Tags         products
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        userId path string true "User ID"
// @Param        productId path string true "Product ID"
// @Param        request body setSubscriptionRequest true "Subscription payload"
// @Success      200 {object} subscriptionResponse
// @Failure      403 {object} map[string]string
// @Router       /users/{userId}/products/{productId}/subscribe [post]
func (h *ProductHandler) SetSubscription(w http.ResponseWriter, r *http.Request) {
	caller, ok := middleware.CallerFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req setSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	targetUserID := chi.URLParam(r, "userId")
	productID := chi.URLParam(r, "productId")
	sub, err := h.svc.SetSubscription(r.Context(), caller, targetUserID, productID, req.Subscripted, req.Renewed)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrForbidden):
			writeError(w, http.StatusForbidden, "superadmin or admin role required")
		case errors.Is(err, repository.ErrProductNotFound):
			writeError(w, http.StatusNotFound, "product not found")
		default:
			writeError(w, http.StatusInternalServerError, "failed to set subscription")
		}
		return
	}
	writeJSON(w, http.StatusOK, subscriptionResponse{
		UserID: sub.UserID, ProductID: sub.ProductID, Subscripted: sub.Subscripted, Renewed: sub.Renewed,
	})
}
