package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	_ "localization-service/internal/models" // referenced by swag annotations
	"localization-service/internal/service"
)

type TranslationHandler struct {
	svc *service.TranslationService
}

func NewTranslationHandler(svc *service.TranslationService) *TranslationHandler {
	return &TranslationHandler{svc: svc}
}

type upsertRequest struct {
	Namespace string `json:"namespace"`
	Key       string `json:"key"`
	Locale    string `json:"locale"`
	Value     string `json:"value"`
}

// Upsert godoc
// @Summary      Create or update a translation (administrator only)
// @Description  (namespace, key, locale) üçlüyü artıq mövcuddursa dəyərini yeniləyir, yoxdursa yaradır.
// @Tags         translations
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body upsertRequest true "Translation payload"
// @Success      200 {object} models.Translation
// @Failure      400 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Router       /translations [post]
func (h *TranslationHandler) Upsert(w http.ResponseWriter, r *http.Request) {
	var req upsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	t, err := h.svc.Upsert(r.Context(), req.Namespace, req.Key, req.Locale, req.Value)
	if err != nil {
		writeValidationError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, t)
}

// List godoc
// @Summary      List translations
// @Description  Filtrlər opsionaldır. Public — auth tələb olunmur (login/xəta səhifələri autentifikasiyadan əvvəl tərcümələrə ehtiyac duya bilər).
// @Tags         translations
// @Produce      json
// @Param        namespace query string false "Filter by namespace"
// @Param        locale query string false "Filter by locale"
// @Param        key query string false "Filter by key"
// @Success      200 {array} models.Translation
// @Router       /translations [get]
func (h *TranslationHandler) List(w http.ResponseWriter, r *http.Request) {
	namespace := r.URL.Query().Get("namespace")
	locale := r.URL.Query().Get("locale")
	key := r.URL.Query().Get("key")

	translations, err := h.svc.List(r.Context(), namespace, locale, key)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list translations")
		return
	}

	writeJSON(w, http.StatusOK, translations)
}

// Map godoc
// @Summary      Get a namespace's translations as a key→value map
// @Description  Bir dəfə yüklənib client tərəfdə birbaşa istifadə edilməsi üçün ən rahat format.
// @Tags         translations
// @Produce      json
// @Param        namespace query string true "Namespace"
// @Param        locale query string true "Locale"
// @Success      200 {object} map[string]string
// @Failure      400 {object} map[string]string
// @Router       /translations/map [get]
func (h *TranslationHandler) Map(w http.ResponseWriter, r *http.Request) {
	namespace := r.URL.Query().Get("namespace")
	locale := r.URL.Query().Get("locale")
	if namespace == "" || locale == "" {
		writeError(w, http.StatusBadRequest, "namespace and locale query params are required")
		return
	}

	result, err := h.svc.Map(r.Context(), namespace, locale)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to build translation map")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// Lookup godoc
// @Summary      Get a single translation, with default-locale fallback
// @Tags         translations
// @Produce      json
// @Param        namespace query string true "Namespace"
// @Param        key query string true "Key"
// @Param        locale query string true "Locale"
// @Success      200 {object} models.Translation
// @Failure      400 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /translations/lookup [get]
func (h *TranslationHandler) Lookup(w http.ResponseWriter, r *http.Request) {
	namespace := r.URL.Query().Get("namespace")
	key := r.URL.Query().Get("key")
	locale := r.URL.Query().Get("locale")
	if namespace == "" || key == "" || locale == "" {
		writeError(w, http.StatusBadRequest, "namespace, key and locale query params are required")
		return
	}

	t, err := h.svc.Get(r.Context(), namespace, key, locale)
	if err != nil {
		if errors.Is(err, service.ErrTranslationNotFound) {
			writeError(w, http.StatusNotFound, "translation not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load translation")
		return
	}

	writeJSON(w, http.StatusOK, t)
}

// Delete godoc
// @Summary      Delete a translation (administrator only)
// @Tags         translations
// @Security     BearerAuth
// @Param        id path string true "Translation ID"
// @Success      204 "No Content"
// @Failure      404 {object} map[string]string
// @Router       /translations/{id} [delete]
func (h *TranslationHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, service.ErrTranslationNotFound) {
			writeError(w, http.StatusNotFound, "translation not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete translation")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListLocales godoc
// @Summary      List locales currently in use
// @Tags         translations
// @Produce      json
// @Success      200 {array} string
// @Router       /locales [get]
func (h *TranslationHandler) ListLocales(w http.ResponseWriter, r *http.Request) {
	locales, err := h.svc.ListLocales(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list locales")
		return
	}

	writeJSON(w, http.StatusOK, locales)
}

func writeValidationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrNamespaceRequired),
		errors.Is(err, service.ErrKeyRequired),
		errors.Is(err, service.ErrLocaleRequired),
		errors.Is(err, service.ErrValueRequired):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "failed to save translation")
	}
}
