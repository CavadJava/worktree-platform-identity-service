package models

import "time"

// Order status is a strictly forward-moving pipeline — a shop (or the
// admin) drives it from Pending through to Delivered; a customer can only
// ever set it to Delivered themselves (confirming receipt), never any
// other stage. StatusOrder defines that sequence; see service.UpdateStatus
// for the transition rule ("no backward moves, no skipping to an earlier
// stage than the current one").
const (
	StatusPending    = "pending"    // sifariş qəbul edildi
	StatusProcessing = "processing" // mağaza hazırlayır
	StatusShipped    = "shipped"    // mağazadan çıxdı, daşıyıcıya verildi
	StatusInTransit  = "in_transit" // yoldadır (məs. gömrük/hava limanı)
	StatusDelivered  = "delivered"  // müştəriyə çatıb — son mərhələ
)

var StatusOrder = []string{StatusPending, StatusProcessing, StatusShipped, StatusInTransit, StatusDelivered}

// StatusIndex returns a status's position in the pipeline, or -1 if it
// isn't a recognized status.
func StatusIndex(status string) int {
	for i, s := range StatusOrder {
		if s == status {
			return i
		}
	}
	return -1
}

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
	ID            string    `json:"id"`
	OrderID       string    `json:"order_id"`
	ProductID     string    `json:"product_id"`
	ProductItemID string    `json:"product_item_id"`
	ProductName   string    `json:"product_name"`
	ItemName      string    `json:"item_name"`
	UnitPrice     float64   `json:"unit_price"`
	Quantity      int       `json:"quantity"`
	CreatedAt     time.Time `json:"created_at"`
}
