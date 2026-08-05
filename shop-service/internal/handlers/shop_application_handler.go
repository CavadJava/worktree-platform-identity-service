package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"shop-service/internal/middleware"
	_ "shop-service/internal/models" // referenced by swag annotations
	"shop-service/internal/roles"
	"shop-service/internal/service"
)

type ShopApplicationHandler struct {
	svc *service.ShopApplicationService
}

func NewShopApplicationHandler(svc *service.ShopApplicationService) *ShopApplicationHandler {
	return &ShopApplicationHandler{svc: svc}
}

type applicationRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type rejectRequest struct {
	Reason string `json:"reason"`
}

// Submit godoc
// @Summary      Apply to open a shop
// @Description  İstənilən login olmuş istifadəçi müraciət göndərə bilər. Müraciət "pending" statusunda yaranır — administrator ya birbaşa rədd edir, ya da formu göndərir (/send-form), bundan sonra mağaza açılır.
// @Tags         shop-applications
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body applicationRequest true "Application payload"
// @Success      201 {object} models.ShopApplication
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Router       /shop-applications [post]
func (h *ShopApplicationHandler) Submit(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req applicationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	app, err := h.svc.Submit(r.Context(), identity.UserID, req.Name, req.Description)
	if err != nil {
		if errors.Is(err, service.ErrApplicationNameMissing) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to submit application")
		return
	}

	writeJSON(w, http.StatusCreated, app)
}

// List godoc
// @Summary      List shop applications (administrator only)
// @Tags         shop-applications
// @Produce      json
// @Security     BearerAuth
// @Param        status query string false "pending | form_sent | approved | rejected"
// @Success      200 {array} models.ShopApplication
// @Failure      403 {object} map[string]string
// @Router       /shop-applications [get]
func (h *ShopApplicationHandler) List(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	apps, err := h.svc.List(r.Context(), status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list applications")
		return
	}
	writeJSON(w, http.StatusOK, apps)
}

// Get godoc
// @Summary      Get a shop application
// @Description  Administrator, ya da müraciəti göndərən istifadəçinin özü baxa bilər.
// @Tags         shop-applications
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Application ID"
// @Success      200 {object} models.ShopApplication
// @Failure      403 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /shop-applications/{id} [get]
func (h *ShopApplicationHandler) Get(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	app, err := h.svc.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrApplicationNotFound) {
			writeError(w, http.StatusNotFound, "application not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load application")
		return
	}

	if identity.Role != roles.RoleAdministrator && identity.UserID != app.ApplicantID {
		writeError(w, http.StatusForbidden, "only the applicant or an administrator can view this application")
		return
	}

	writeJSON(w, http.StatusOK, app)
}

// SendForm godoc
// @Summary      Send the shop form (administrator only)
// @Description  Müraciəti irəli aparmaq qərarı: müvəqqəti mağaza yaradır, müraciət sahibini onun admin(4) səviyyəsinə təyin edir ki, mağaza məlumatlarını özü doldursun (PUT /shops/{id}). Mağaza yekun təsdiqə qədər (approve) public siyahılarda görünmür.
// @Tags         shop-applications
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Application ID"
// @Success      200 {object} models.ShopApplication
// @Failure      400 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /shop-applications/{id}/send-form [post]
func (h *ShopApplicationHandler) SendForm(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	app, err := h.svc.SendForm(r.Context(), identity.UserID, id)
	writeDecisionResult(w, app, err)
}

// Approve godoc
// @Summary      Approve a shop application (administrator only)
// @Description  Yalnız "form_sent" statusundan çağırıla bilər. Müvəqqəti mağazanı daimi edir (public siyahılarda görünməyə başlayır).
// @Tags         shop-applications
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Application ID"
// @Success      200 {object} models.ShopApplication
// @Failure      400 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /shop-applications/{id}/approve [post]
func (h *ShopApplicationHandler) Approve(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	app, err := h.svc.Approve(r.Context(), identity.UserID, id)
	writeDecisionResult(w, app, err)
}

// Reject godoc
// @Summary      Reject a shop application (administrator only)
// @Description  "pending" statusundan (heç bir mağaza yaranmayıb) və ya "form_sent" statusundan (müvəqqəti mağaza silinir, təyinat geri alınır) çağırıla bilər.
// @Tags         shop-applications
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Application ID"
// @Param        request body rejectRequest false "Reject payload"
// @Success      200 {object} models.ShopApplication
// @Failure      400 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /shop-applications/{id}/reject [post]
func (h *ShopApplicationHandler) Reject(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req rejectRequest
	_ = json.NewDecoder(r.Body).Decode(&req) // reason is optional

	id := chi.URLParam(r, "id")
	app, err := h.svc.Reject(r.Context(), identity.UserID, id, req.Reason)
	writeDecisionResult(w, app, err)
}

func writeDecisionResult(w http.ResponseWriter, app interface{}, err error) {
	if err != nil {
		switch {
		case errors.Is(err, service.ErrApplicationNotFound):
			writeError(w, http.StatusNotFound, "application not found")
		case errors.Is(err, service.ErrApplicationAlreadyDecided), errors.Is(err, service.ErrApplicationNotFormSent):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "failed to process decision")
		}
		return
	}
	writeJSON(w, http.StatusOK, app)
}
