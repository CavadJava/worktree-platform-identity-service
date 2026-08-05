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

type welcomeNotificationRequest struct {
	Type     string `json:"type"`
	To       string `json:"to"`
	FullName string `json:"full_name"`
}

// SendWelcome notifies notification-service that a new user registered.
// Failures are returned to the caller, who decides whether they should block registration.
func (c *NotificationClient) SendWelcome(ctx context.Context, email, fullName string) error {
	body, err := json.Marshal(welcomeNotificationRequest{
		Type:     "welcome_email",
		To:       email,
		FullName: fullName,
	})
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
