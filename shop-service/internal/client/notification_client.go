package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type NotificationClient struct {
	baseURL string
	http    *http.Client
}

func NewNotificationClient(baseURL string) *NotificationClient {
	return &NotificationClient{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

type notificationRequest struct {
	Type     string `json:"type"`
	To       string `json:"to"`
	FullName string `json:"full_name"`
	Message  string `json:"message,omitempty"`
}

// Send notifies notification-service about a shop-application status change.
// Failures are logged by the caller but never block the underlying decision
// (approve/reject/send-form already happened by the time this is called).
func (c *NotificationClient) Send(ctx context.Context, notificationType, email, fullName, message string) error {
	body, err := json.Marshal(notificationRequest{Type: notificationType, To: email, FullName: fullName, Message: message})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/notifications", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("notification-service returned status %d", resp.StatusCode)
	}
	return nil
}
