package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"log-service/internal/models"
	"log-service/internal/repository"
)

const (
	defaultLimit = 100
	maxLimit     = 500
)

type LogHandler struct {
	repo *repository.LogRepository
}

func NewLogHandler(repo *repository.LogRepository) *LogHandler {
	return &LogHandler{repo: repo}
}

type logRequest struct {
	Service    string `json:"service"`
	Level      string `json:"level"`
	Method     string `json:"method,omitempty"`
	Path       string `json:"path,omitempty"`
	Status     int    `json:"status,omitempty"`
	DurationMs int64  `json:"duration_ms,omitempty"`
	Message    string `json:"message,omitempty"`
}

// Create godoc
// @Summary      Ingest a log entry
// @Description  Digər servislər öz request/error loglarını buraya göndərir (fire-and-forget).
// @Tags         logs
// @Accept       json
// @Produce      json
// @Param        request body logRequest true "Log entry"
// @Success      201 {object} models.LogEntry
// @Failure      400 {object} map[string]string
// @Router       /logs [post]
func (h *LogHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req logRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Service == "" {
		writeError(w, http.StatusBadRequest, "service is required")
		return
	}
	level := req.Level
	switch level {
	case "":
		level = "info"
	case "info", "warn", "error":
	default:
		writeError(w, http.StatusBadRequest, "level must be one of: info, warn, error")
		return
	}

	e := &models.LogEntry{
		ID:         uuid.NewString(),
		Service:    req.Service,
		Level:      level,
		Method:     req.Method,
		Path:       req.Path,
		Status:     req.Status,
		DurationMs: req.DurationMs,
		Message:    req.Message,
		CreatedAt:  time.Now().UTC(),
	}
	if err := h.repo.Create(r.Context(), e); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to store log entry")
		return
	}
	writeJSON(w, http.StatusCreated, e)
}

// List godoc
// @Summary      Query logs
// @Description  Servis, səviyyə, path və tarix aralığı üzrə filtr. Ən yenilər əvvəldə.
// @Tags         logs
// @Produce      json
// @Param        service query string false "Service name"
// @Param        level   query string false "info | warn | error"
// @Param        path    query string false "Path substring (ILIKE)"
// @Param        from    query string false "RFC3339, e.g. 2026-08-05T00:00:00Z"
// @Param        to      query string false "RFC3339"
// @Param        limit   query int    false "Default 100, max 500"
// @Success      200 {array} models.LogEntry
// @Failure      400 {object} map[string]string
// @Router       /logs [get]
func (h *LogHandler) List(w http.ResponseWriter, r *http.Request) {
	qp := r.URL.Query()
	f := repository.LogFilter{
		Service: qp.Get("service"),
		Level:   qp.Get("level"),
		Path:    qp.Get("path"),
		Limit:   defaultLimit,
	}
	if v := qp.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			writeError(w, http.StatusBadRequest, "limit must be a positive integer")
			return
		}
		if n > maxLimit {
			n = maxLimit
		}
		f.Limit = n
	}
	if v := qp.Get("from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "from must be RFC3339")
			return
		}
		f.From = &t
	}
	if v := qp.Get("to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "to must be RFC3339")
			return
		}
		f.To = &t
	}

	entries, err := h.repo.List(r.Context(), f)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query logs")
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

// ListServices godoc
// @Summary      List services that have sent logs
// @Tags         logs
// @Produce      json
// @Success      200 {array} string
// @Router       /logs/services [get]
func (h *LogHandler) ListServices(w http.ResponseWriter, r *http.Request) {
	services, err := h.repo.ListServices(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list services")
		return
	}
	writeJSON(w, http.StatusOK, services)
}
