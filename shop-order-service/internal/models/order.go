package models

import "time"

const (
	StatusPending = "pending"
)

type Order struct {
	ID          string      `json:"id"`
	OrderNumber string      `json:"order_number"`
	UserID      string      `json:"user_id"`
	ShopID      string      `json:"shop_id"`
	Status      string      `json:"status"`
	TotalAmount float64     `json:"total_amount"`
	Items       []OrderItem `json:"items,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

type OrderItem struct {
	ID          string    `json:"id"`
	OrderID     string    `json:"order_id"`
	ProductID   string    `json:"product_id"`
	ProductName string    `json:"product_name"`
	UnitPrice   float64   `json:"unit_price"`
	Quantity    int       `json:"quantity"`
	CreatedAt   time.Time `json:"created_at"`
}
