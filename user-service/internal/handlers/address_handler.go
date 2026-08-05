package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"user-service/internal/middleware"
	_ "user-service/internal/models" // referenced by swag annotations
	"user-service/internal/service"
)

type AddressHandler struct {
	svc *service.AddressService
}

func NewAddressHandler(svc *service.AddressService) *AddressHandler {
	return &AddressHandler{svc: svc}
}

type addressRequest struct {
	Title       string `json:"title"`
	FullAddress string `json:"full_address"`
	City        string `json:"city,omitempty"`
	Phone       string `json:"phone,omitempty"`
	IsDefault   bool   `json:"is_default"`
}

// Create godoc
// @Summary      Add an address to the current user's address book
// @Description  is_default=true göndərilsə əvvəlki default sıfırlanır — hər istifadəçidə maksimum bir default ünvan olur.
// @Tags         addresses
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body addressRequest true "Address payload"
// @Success      201 {object} models.Address
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Router       /addresses [post]
func (h *AddressHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req addressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.FullAddress) == "" {
		writeError(w, http.StatusBadRequest, "title and full_address are required")
		return
	}

	address, err := h.svc.Create(r.Context(), userID, req.Title, req.FullAddress, req.City, req.Phone, req.IsDefault)
	if err != nil {
		writeAddressError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, address)
}

// List godoc
// @Summary      List the current user's addresses
// @Description  Default ünvan birinci gəlir.
// @Tags         addresses
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} models.Address
// @Failure      401 {object} map[string]string
// @Router       /addresses [get]
func (h *AddressHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	addresses, err := h.svc.List(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list addresses")
		return
	}
	writeJSON(w, http.StatusOK, addresses)
}

// Update godoc
// @Summary      Update one of the current user's addresses
// @Tags         addresses
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Address ID"
// @Param        request body addressRequest true "Address payload"
// @Success      200 {object} models.Address
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /addresses/{id} [put]
func (h *AddressHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")

	var req addressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.FullAddress) == "" {
		writeError(w, http.StatusBadRequest, "title and full_address are required")
		return
	}

	address, err := h.svc.Update(r.Context(), userID, id, req.Title, req.FullAddress, req.City, req.Phone, req.IsDefault)
	if err != nil {
		writeAddressError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, address)
}

// Delete godoc
// @Summary      Delete one of the current user's addresses
// @Tags         addresses
// @Security     BearerAuth
// @Param        id path string true "Address ID"
// @Success      204 "No Content"
// @Failure      401 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /addresses/{id} [delete]
func (h *AddressHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	if err := h.svc.Delete(r.Context(), userID, id); err != nil {
		writeAddressError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeAddressError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrAddressNotFound):
		writeError(w, http.StatusNotFound, "address not found")
	case errors.Is(err, service.ErrAddressForbidden):
		writeError(w, http.StatusForbidden, "address belongs to another user")
	default:
		writeError(w, http.StatusInternalServerError, "failed to process request")
	}
}
