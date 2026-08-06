package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// PaymentClient talks to payment-service. Called synchronously right after
// an order is created (not fire-and-forget in a goroutine): the shop's
// balance should reflect the sale in the same instant the order is placed,
// not after an eventual-consistency lag. A failure here is still non-fatal
// to order creation though — the caller logs and swallows it, same "never
// let a secondary concern break the primary write" philosophy notification
// fan-out already uses elsewhere in this codebase (see shop-product-service).
type PaymentClient struct {
	baseURL string
	http    *http.Client
}

func NewPaymentClient(baseURL string) *PaymentClient {
	return &PaymentClient{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

type PaymentItemInput struct {
	ProductID   string
	ProductName string
	Quantity    int
	UnitPrice   float64
}

type paymentItemPayload struct {
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
}

type paymentPayload struct {
	OrderID string               `json:"order_id"`
	UserID  string               `json:"user_id"`
	ShopID  string               `json:"shop_id"`
	Amount  float64              `json:"amount"`
	Items   []paymentItemPayload `json:"items"`
}

func (c *PaymentClient) RecordPayment(ctx context.Context, orderID, userID, shopID string, amount float64, items []PaymentItemInput) error {
	payloadItems := make([]paymentItemPayload, 0, len(items))
	for _, it := range items {
		payloadItems = append(payloadItems, paymentItemPayload{
			ProductID:   it.ProductID,
			ProductName: it.ProductName,
			Quantity:    it.Quantity,
			UnitPrice:   it.UnitPrice,
		})
	}

	body, err := json.Marshal(paymentPayload{OrderID: orderID, UserID: userID, ShopID: shopID, Amount: amount, Items: payloadItems})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/payments", bytes.NewReader(body))
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
		return fmt.Errorf("payment-service returned status %d", resp.StatusCode)
	}
	return nil
}
