package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

type NotificationHandler struct{}

func NewNotificationHandler() *NotificationHandler {
	return &NotificationHandler{}
}

type sendNotificationRequest struct {
	Type     string `json:"type"`
	To       string `json:"to"`
	FullName string `json:"full_name"`
	Message  string `json:"message,omitempty"`
}

type sendNotificationResponse struct {
	Status string `json:"status"`
}

// Send godoc
// @Summary      Send a notification (mocked)
// @Description  Bildirişi real olaraq göndərmir — konsola loglayır (mock). Real provider (SMTP/SMS) əlavə etmək üçün buraya inteqrasiya edilə bilər.
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
	if req.To == "" {
		writeError(w, http.StatusBadRequest, "to is required")
		return
	}
	if req.Type == "" {
		writeError(w, http.StatusBadRequest, "type is required")
		return
	}

	// Mocked send: no real email/SMS provider wired up, just log what would go out.
	log.Printf("[MOCK NOTIFICATION] type=%s to=%s full_name=%q message=%q", req.Type, req.To, req.FullName, req.Message)

	writeJSON(w, http.StatusAccepted, sendNotificationResponse{Status: "sent"})
}
