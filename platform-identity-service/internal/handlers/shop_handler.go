package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"platform-identity-service/internal/repository"
	"platform-identity-service/internal/service"
)

type ShopHandler struct {
	svc *service.ShopService
}

func NewShopHandler(svc *service.ShopService) *ShopHandler {
	return &ShopHandler{svc: svc}
}

type createShopRequest struct {
	Name string `json:"name"`
}

type shopResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

// Create godoc
// @Summary      Register a new shop
// @Tags         shops
// @Accept       json
// @Produce      json
// @Param        request body createShopRequest true "Shop payload"
// @Success      201 {object} shopResponse
// @Failure      400 {object} map[string]string
// @Router       /shops [post]
func (h *ShopHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createShopRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	s, err := h.svc.Create(r.Context(), req.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create shop")
		return
	}

	writeJSON(w, http.StatusCreated, shopResponse{ID: s.ID, Name: s.Name, CreatedAt: s.CreatedAt.Format(timeFormat)})
}

// List godoc
// @Summary      List all shops
// @Tags         shops
// @Produce      json
// @Success      200 {array} shopResponse
// @Router       /shops [get]
func (h *ShopHandler) List(w http.ResponseWriter, r *http.Request) {
	shops, err := h.svc.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list shops")
		return
	}

	response := make([]shopResponse, len(shops))
	for i, s := range shops {
		response[i] = shopResponse{ID: s.ID, Name: s.Name, CreatedAt: s.CreatedAt.Format(timeFormat)}
	}
	writeJSON(w, http.StatusOK, response)
}

// Get godoc
// @Summary      Get a shop by id
// @Tags         shops
// @Produce      json
// @Param        id path string true "Shop ID"
// @Success      200 {object} shopResponse
// @Failure      404 {object} map[string]string
// @Router       /shops/{id} [get]
func (h *ShopHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	s, err := h.svc.Get(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrShopNotFound):
			writeError(w, http.StatusNotFound, "shop not found")
		default:
			writeError(w, http.StatusInternalServerError, "failed to get shop")
		}
		return
	}
	writeJSON(w, http.StatusOK, shopResponse{ID: s.ID, Name: s.Name, CreatedAt: s.CreatedAt.Format(timeFormat)})
}

const timeFormat = "2006-01-02T15:04:05Z07:00"
