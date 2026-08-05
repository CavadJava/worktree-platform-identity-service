package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

var ErrUnauthorized = errors.New("unauthorized")

type AuthorizationClient struct {
	baseURL string
	http    *http.Client
}

func NewAuthorizationClient(baseURL string) *AuthorizationClient {
	return &AuthorizationClient{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

// Authorize forwards the incoming Authorization header to authorization-service
// and returns the authenticated user's ID if the token is valid.
func (c *AuthorizationClient) Authorize(ctx context.Context, authHeader string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/authorize", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", authHeader)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return "", ErrUnauthorized
	}
	if resp.StatusCode != http.StatusOK {
		return "", errors.New("authorization-service returned an unexpected status")
	}

	var envelope struct {
		Data struct {
			UserID string `json:"user_id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return "", err
	}
	return envelope.Data.UserID, nil
}
