package handlers

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"shop-service/internal/middleware"
	_ "shop-service/internal/models" // referenced by swag annotations
	"shop-service/internal/service"
)

type SubscriptionHandler struct {
	svc *service.SubscriptionService
}

func NewSubscriptionHandler(svc *service.SubscriptionService) *SubscriptionHandler {
	return &SubscriptionHandler{svc: svc}
}

// Subscribe godoc
// @Summary      Subscribe to a shop
// @Tags         subscriptions
// @Security     BearerAuth
// @Param        id path string true "Shop ID"
// @Success      204 "No Content"
// @Failure      404 {object} map[string]string
// @Router       /shops/{id}/subscribe [post]
func (h *SubscriptionHandler) Subscribe(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	shopID := chi.URLParam(r, "id")
	if err := h.svc.Subscribe(r.Context(), identity.UserID, shopID); err != nil {
		if errors.Is(err, service.ErrShopNotFound) {
			writeError(w, http.StatusNotFound, "shop not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to subscribe")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Unsubscribe godoc
// @Summary      Unsubscribe from a shop
// @Tags         subscriptions
// @Security     BearerAuth
// @Param        id path string true "Shop ID"
// @Success      204 "No Content"
// @Router       /shops/{id}/subscribe [delete]
func (h *SubscriptionHandler) Unsubscribe(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	shopID := chi.URLParam(r, "id")
	if err := h.svc.Unsubscribe(r.Context(), identity.UserID, shopID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to unsubscribe")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// List godoc
// @Summary      List shops I'm subscribed to
// @Tags         subscriptions
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} models.Shop
// @Router       /subscriptions [get]
func (h *SubscriptionHandler) List(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	shops, err := h.svc.List(r.Context(), identity.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list subscriptions")
		return
	}

	writeJSON(w, http.StatusOK, shops)
}

// ListSubscribers godoc
// @Summary      List a shop's subscribers
// @Description  chat(1)+ səviyyəli mağaza əməkdaşı (öz mağazasında) və ya administrator.
// @Tags         subscriptions
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Shop ID"
// @Success      200 {array} models.Subscriber
// @Failure      403 {object} map[string]string
// @Router       /shops/{id}/subscribers [get]
func (h *SubscriptionHandler) ListSubscribers(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	shopID := chi.URLParam(r, "id")
	subscribers, err := h.svc.ListSubscribers(r.Context(), toIdentity(identity), shopID)
	if err != nil {
		if errors.Is(err, service.ErrForbidden) {
			writeError(w, http.StatusForbidden, "insufficient permission: must belong to this shop, or be an administrator")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to list subscribers")
		return
	}

	writeJSON(w, http.StatusOK, subscribers)
}
