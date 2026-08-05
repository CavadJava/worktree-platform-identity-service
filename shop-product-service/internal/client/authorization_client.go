package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

var ErrUnauthorized = errors.New("unauthorized")

type Identity struct {
	UserID        string  `json:"user_id"`
	Role          string  `json:"role"`
	ShopID        *string `json:"shop_id,omitempty"`
	ShopRoleLevel int     `json:"shop_role_level"`
}

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

func (c *AuthorizationClient) Authorize(ctx context.Context, authHeader string) (*Identity, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/authorize", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", authHeader)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, ErrUnauthorized
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("authorization-service returned an unexpected status")
	}

	var envelope struct {
		Data Identity `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}
