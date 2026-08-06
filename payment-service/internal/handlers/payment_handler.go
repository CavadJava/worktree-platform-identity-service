package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"payment-service/internal/client"
	"payment-service/internal/middleware"
	_ "payment-service/internal/models" // referenced by swag annotations
	"payment-service/internal/service"
)

type PaymentHandler struct {
	svc *service.PaymentService
}

func NewPaymentHandler(svc *service.PaymentService) *PaymentHandler {
	return &PaymentHandler{svc: svc}
}

type paymentItemRequest struct {
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
}

type paymentRequest struct {
	OrderID string               `json:"order_id"`
	UserID  string               `json:"user_id"`
	ShopID  string               `json:"shop_id"`
	Amount  float64              `json:"amount"`
	Items   []paymentItemRequest `json:"items"`
}

// Create godoc
// @Summary      Record a completed payment for an order
// @Description  Internal, service-to-service call (no auth — same convention as notification-service's POST /notifications) made once by shop-order-service right after it creates an order. Credits the shop's balance atomically.
// @Tags         payments
// @Accept       json
// @Produce      json
// @Param        request body paymentRequest true "Payment payload"
// @Success      201 {object} models.Payment
// @Failure      400 {object} map[string]string
// @Failure      409 {object} map[string]string
// @Router       /payments [post]
func (h *PaymentHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req paymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.OrderID == "" || req.UserID == "" || req.ShopID == "" {
		writeError(w, http.StatusBadRequest, "order_id, user_id and shop_id are required")
		return
	}

	items := make([]service.ItemInput, 0, len(req.Items))
	for _, it := range req.Items {
		items = append(items, service.ItemInput{
			ProductID:   it.ProductID,
			ProductName: it.ProductName,
			Quantity:    it.Quantity,
			UnitPrice:   it.UnitPrice,
		})
	}

	payment, err := h.svc.Create(r.Context(), req.OrderID, req.UserID, req.ShopID, req.Amount, items)
	if err != nil {
		writePaymentError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, payment)
}

// ListByShop godoc
// @Summary      List a shop's payments
// @Description  chat(1)+ səviyyəli mağaza əməkdaşı (öz mağazasında) və ya administrator.
// @Tags         payments
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Shop ID"
// @Success      200 {array} models.Payment
// @Failure      403 {object} map[string]string
// @Router       /shops/{id}/payments [get]
func (h *PaymentHandler) ListByShop(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	shopID := chi.URLParam(r, "id")
	payments, err := h.svc.ListByShop(r.Context(), toIdentity(identity), shopID)
	if err != nil {
		writePaymentError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, payments)
}

// GetBalance godoc
// @Summary      Get a shop's (temporary) balance
// @Description  chat(1)+ səviyyəli mağaza əməkdaşı (öz mağazasında) və ya administrator. Bu bakiyə hələ ki yalnız payment-service daxilində yaşayır — real mağaza bank hesabına köçürmə gələcək bir funksiyadır.
// @Tags         payments
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Shop ID"
// @Success      200 {object} models.ShopBalance
// @Failure      403 {object} map[string]string
// @Router       /shops/{id}/balance [get]
func (h *PaymentHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	shopID := chi.URLParam(r, "id")
	balance, err := h.svc.GetBalance(r.Context(), toIdentity(identity), shopID)
	if err != nil {
		writePaymentError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, balance)
}

func toIdentity(identity *client.Identity) service.Identity {
	return service.Identity{
		UserID:        identity.UserID,
		Role:          identity.Role,
		ShopID:        identity.ShopID,
		ShopRoleLevel: identity.ShopRoleLevel,
	}
}

func writePaymentError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrItemsRequired), errors.Is(err, service.ErrInvalidAmount):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrPaymentAlreadyExists):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, service.ErrForbidden):
		writeError(w, http.StatusForbidden, "insufficient permission: must belong to this shop (chat(1)+), or be an administrator")
	default:
		writeError(w, http.StatusInternalServerError, "failed to process request")
	}
}
