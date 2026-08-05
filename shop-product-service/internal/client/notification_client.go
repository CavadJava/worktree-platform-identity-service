package client

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"
)

// NotificationClient talks to notification-service. Sends are best-effort:
// a failure is the caller's to log, never to propagate — a missed
// notification must not fail the product operation that triggered it.
type NotificationClient struct {
	baseURL string
	http    *http.Client
}

func NewNotificationClient(baseURL string) *NotificationClient {
	return &NotificationClient{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 3 * time.Second},
	}
}

type notificationPayload struct {
	Type    string `json:"type"`
	UserID  string `json:"user_id"`
	Message string `json:"message"`
}

func (c *NotificationClient) Send(notificationType, userID, message string) error {
	body, err := json.Marshal(notificationPayload{Type: notificationType, UserID: userID, Message: message})
	if err != nil {
		return err
	}
	resp, err := c.http.Post(c.baseURL+"/api/v1/notifications", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
