package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"shop-order-service/internal/client"
	"shop-order-service/internal/middleware"
	_ "shop-order-service/internal/models" // referenced by swag annotations
	"shop-order-service/internal/service"
)

type OrderHandler struct {
	svc *service.OrderService
}

func NewOrderHandler(svc *service.OrderService) *OrderHandler {
	return &OrderHandler{svc: svc}
}

type orderItemRequest struct {
	ProductItemID string `json:"product_item_id"`
	Quantity      int    `json:"quantity"`
}

type createOrderRequest struct {
	ShopID string             `json:"shop_id"`
	Items  []orderItemRequest `json:"items"`
}

// Create godoc
// @Summary      Create an order
// @Description  İstənilən login olmuş istifadəçi bir mağazadan bir və ya bir neçə məhsul VARİANTI (product_item_id, məs. "30x60 1 qat") seçib sifariş yarada bilər. Qiymət (endirimli olsa endirim qiyməti) və ad sifariş anında "şəkil" kimi saxlanılır (sonradan dəyişsə belə sifariş dəyişmir).
// @Tags         orders
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body createOrderRequest true "Order payload"
// @Success      201 {object} models.Order
// @Failure      400 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /orders [post]
func (h *OrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req createOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ShopID == "" {
		writeError(w, http.StatusBadRequest, "shop_id is required")
		return
	}

	items := make([]service.ItemInput, len(req.Items))
	for i, it := range req.Items {
		items[i] = service.ItemInput{ProductItemID: it.ProductItemID, Quantity: it.Quantity}
	}

	order, err := h.svc.Create(r.Context(), identity.UserID, req.ShopID, items)
	if err != nil {
		writeOrderError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, order)
}

// ListMine godoc
// @Summary      List my orders
// @Tags         orders
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} models.Order
// @Router       /orders [get]
func (h *OrderHandler) ListMine(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	orders, err := h.svc.ListMine(r.Context(), identity.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list orders")
		return
	}

	writeJSON(w, http.StatusOK, orders)
}

// Get godoc
// @Summary      Get an order (with items)
// @Description  Sifarişi verən istifadəçi, mağazanın add-product(2)+ əməkdaşı, ya da administrator baxa bilər.
// @Tags         orders
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Order ID"
// @Success      200 {object} models.Order
// @Failure      403 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /orders/{id} [get]
func (h *OrderHandler) Get(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	order, err := h.svc.Get(r.Context(), toIdentity(identity), id)
	if err != nil {
		writeOrderError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, order)
}

// ListForShop godoc
// @Summary      List a shop's incoming orders
// @Description  Mağazanın add-product(2)+ səviyyəli əməkdaşı, ya da administrator.
// @Tags         orders
// @Produce      json
// @Security     BearerAuth
// @Param        shop_id path string true "Shop ID"
// @Success      200 {array} models.Order
// @Failure      403 {object} map[string]string
// @Router       /shops/{shop_id}/orders [get]
func (h *OrderHandler) ListForShop(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	shopID := chi.URLParam(r, "shop_id")
	orders, err := h.svc.ListForShop(r.Context(), toIdentity(identity), shopID)
	if err != nil {
		writeOrderError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, orders)
}

func writeOrderError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrItemsRequired), errors.Is(err, service.ErrInvalidQuantity), errors.Is(err, service.ErrProductShopMismatch):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrOrderNotFound):
		writeError(w, http.StatusNotFound, "order not found")
	case errors.Is(err, service.ErrProductItemNotFound):
		writeError(w, http.StatusNotFound, "product item not found")
	case errors.Is(err, service.ErrShopNotFound):
		writeError(w, http.StatusNotFound, "shop not found")
	case errors.Is(err, service.ErrUserNotFound):
		writeError(w, http.StatusNotFound, "user not found")
	case errors.Is(err, service.ErrForbidden):
		writeError(w, http.StatusForbidden, "not authorized to view this order")
	default:
		writeError(w, http.StatusInternalServerError, "failed to process request")
	}
}

func toIdentity(identity *client.Identity) service.Identity {
	return service.Identity{
		UserID:        identity.UserID,
		Role:          identity.Role,
		ShopID:        identity.ShopID,
		ShopRoleLevel: identity.ShopRoleLevel,
	}
}
