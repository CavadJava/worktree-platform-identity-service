package models

import "time"

// ProductItem is a priced/stocked variant of a Product — e.g. product
// "Bayraq" (Flag) can have items "30x60 1 qat" and "100x100 2 qat", each
// with its own price and stock.
type ProductItem struct {
	ID            string    `json:"id"`
	ProductID     string    `json:"product_id"`
	Name          string    `json:"name"`
	Price         float64   `json:"price"`
	Stock         int       `json:"stock"`
	IsDiscounted  bool      `json:"is_discounted"`
	DiscountPrice *float64  `json:"discount_price,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
