package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"notification-service/internal/middleware"
	"notification-service/internal/models"
	"notification-service/internal/repository"
)

const (
	defaultInboxLimit = 50
	maxInboxLimit     = 200
)

type NotificationHandler struct {
	repo *repository.NotificationRepository
}

func NewNotificationHandler(repo *repository.NotificationRepository) *NotificationHandler {
	return &NotificationHandler{repo: repo}
}

type sendNotificationRequest struct {
	Type     string `json:"type"`
	To       string `json:"to"`
	UserID   string `json:"user_id,omitempty"` // verilsə bildiriş inbox-da saxlanılır
	FullName string `json:"full_name"`
	Message  string `json:"message,omitempty"`
}

type sendNotificationResponse struct {
	Status string `json:"status"`
	Stored bool   `json:"stored"`
}

// Send godoc
// @Summary      Send a notification
// @Description  Real provayder yoxdur — konsola loglanır (mock). `user_id` verilsə bildiriş həmçinin istifadəçinin inbox-unda saxlanılır və GET /notifications ilə oxuna bilir.
// @Tags         notifications
// @Accept       json
// @Produce      json
// @Param        request body sendNotificationRequest true "Notification payload"
// @Success      202 {object} sendNotificationResponse
// @Failure      400 {object} map[string]string
// @Router       /notifications [post]
func (h *NotificationHandler) Send(w http.ResponseWriter, r *http.Request) {
	var req sendNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.To = strings.TrimSpace(req.To)
	if req.To == "" && req.UserID == "" {
		writeError(w, http.StatusBadRequest, "to or user_id is required")
		return
	}
	if req.Type == "" {
		writeError(w, http.StatusBadRequest, "type is required")
		return
	}

	// Mocked send: no real email/SMS provider wired up, just log what would go out.
	log.Printf("[MOCK NOTIFICATION] type=%s to=%s user_id=%s full_name=%q message=%q", req.Type, req.To, req.UserID, req.FullName, req.Message)

	stored := false
	if req.UserID != "" {
		n := &models.Notification{
			ID:        uuid.NewString(),
			UserID:    req.UserID,
			Type:      req.Type,
			Message:   req.Message,
			CreatedAt: time.Now().UTC(),
		}
		if err := h.repo.Create(r.Context(), n); err != nil {
			log.Printf("failed to store notification for user %s: %v", req.UserID, err)
		} else {
			stored = true
		}
	}

	writeJSON(w, http.StatusAccepted, sendNotificationResponse{Status: "sent", Stored: stored})
}

// List godoc
// @Summary      List the current user's notifications (inbox)
// @Tags         notifications
// @Produce      json
// @Security     BearerAuth
// @Param        limit query int false "Default 50, max 200"
// @Success      200 {array} models.Notification
// @Failure      401 {object} map[string]string
// @Router       /notifications [get]
func (h *NotificationHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	limit := defaultInboxLimit
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			writeError(w, http.StatusBadRequest, "limit must be a positive integer")
			return
		}
		if n > maxInboxLimit {
			n = maxInboxLimit
		}
		limit = n
	}

	notifications, err := h.repo.ListByUser(r.Context(), userID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list notifications")
		return
	}
	writeJSON(w, http.StatusOK, notifications)
}

// UnreadCount godoc
// @Summary      Count the current user's unread notifications
// @Tags         notifications
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]int
// @Failure      401 {object} map[string]string
// @Router       /notifications/unread-count [get]
func (h *NotificationHandler) UnreadCount(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	count, err := h.repo.CountUnread(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to count notifications")
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"unread": count})
}

// MarkRead godoc
// @Summary      Mark one of the current user's notifications as read
// @Tags         notifications
// @Security     BearerAuth
// @Param        id path string true "Notification ID"
// @Success      204 "No Content"
// @Failure      401 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /notifications/{id}/read [put]
func (h *NotificationHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := h.repo.MarkRead(r.Context(), userID, chi.URLParam(r, "id")); err != nil {
		if errors.Is(err, repository.ErrNotificationNotFound) {
			writeError(w, http.StatusNotFound, "notification not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to mark notification")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// MarkAllRead godoc
// @Summary      Mark all of the current user's notifications as read
// @Tags         notifications
// @Security     BearerAuth
// @Success      204 "No Content"
// @Failure      401 {object} map[string]string
// @Router       /notifications/read-all [put]
func (h *NotificationHandler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := h.repo.MarkAllRead(r.Context(), userID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to mark notifications")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
