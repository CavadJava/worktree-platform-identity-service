package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"shop-service/internal/middleware"
	_ "shop-service/internal/models" // referenced by swag annotations
	"shop-service/internal/service"
)

type CouponHandler struct {
	svc *service.CouponService
}

func NewCouponHandler(svc *service.CouponService) *CouponHandler {
	return &CouponHandler{svc: svc}
}

type couponRequest struct {
	Code            string `json:"code"`
	Title           string `json:"title"`
	DiscountPercent int    `json:"discount_percent"`
	ValidUntil      string `json:"valid_until,omitempty"` // RFC3339, boş = müddətsiz
}

// Create godoc
// @Summary      Create a coupon (administrator only)
// @Tags         coupons
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body couponRequest true "Coupon payload"
// @Success      201 {object} models.Coupon
// @Failure      400 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Failure      409 {object} map[string]string
// @Router       /coupons [post]
func (h *CouponHandler) Create(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req couponRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.Code) == "" || strings.TrimSpace(req.Title) == "" {
		writeError(w, http.StatusBadRequest, "code and title are required")
		return
	}

	var validUntil *time.Time
	if req.ValidUntil != "" {
		t, err := time.Parse(time.RFC3339, req.ValidUntil)
		if err != nil {
			writeError(w, http.StatusBadRequest, "valid_until must be RFC3339")
			return
		}
		validUntil = &t
	}

	coupon, err := h.svc.Create(r.Context(), toIdentity(identity), req.Code, req.Title, req.DiscountPercent, validUntil)
	if err != nil {
		writeCouponError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, coupon)
}

// List godoc
// @Summary      List active coupons
// @Tags         coupons
// @Produce      json
// @Success      200 {array} models.Coupon
// @Router       /coupons [get]
func (h *CouponHandler) List(w http.ResponseWriter, r *http.Request) {
	coupons, err := h.svc.ListActive(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list coupons")
		return
	}
	writeJSON(w, http.StatusOK, coupons)
}

// Claim godoc
// @Summary      Claim a coupon for the current user
// @Description  İdempotentdir — artıq götürülmüş kuponu yenidən götürmək xəta deyil.
// @Tags         coupons
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Coupon ID"
// @Success      200 {object} models.Coupon
// @Failure      400 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /coupons/{id}/claim [post]
func (h *CouponHandler) Claim(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	coupon, err := h.svc.Claim(r.Context(), identity.UserID, id)
	if err != nil {
		writeCouponError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, coupon)
}

// MyCoupons godoc
// @Summary      List the current user's claimed coupons
// @Tags         coupons
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} models.UserCoupon
// @Router       /my-coupons [get]
func (h *CouponHandler) MyCoupons(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	coupons, err := h.svc.MyCoupons(r.Context(), identity.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list claimed coupons")
		return
	}
	writeJSON(w, http.StatusOK, coupons)
}

// Delete godoc
// @Summary      Delete a coupon (administrator only)
// @Tags         coupons
// @Security     BearerAuth
// @Param        id path string true "Coupon ID"
// @Success      204 "No Content"
// @Failure      403 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /coupons/{id} [delete]
func (h *CouponHandler) Delete(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	if err := h.svc.Delete(r.Context(), toIdentity(identity), id); err != nil {
		writeCouponError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeCouponError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrCouponNotFound):
		writeError(w, http.StatusNotFound, "coupon not found")
	case errors.Is(err, service.ErrCouponCodeExists):
		writeError(w, http.StatusConflict, "coupon code already exists")
	case errors.Is(err, service.ErrCouponExpired), errors.Is(err, service.ErrInvalidDiscount):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrForbidden):
		writeError(w, http.StatusForbidden, "administrator role required")
	default:
		writeError(w, http.StatusInternalServerError, "failed to process request")
	}
}
