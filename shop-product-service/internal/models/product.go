package models

import "time"

type Product struct {
	ID            string    `json:"id"`
	ShopID        string    `json:"shop_id"`
	Name          string    `json:"name"`
	Description   string    `json:"description,omitempty"`
	Price         float64   `json:"price"`
	Stock         int       `json:"stock"`
	ProductTypeID *string   `json:"product_type_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
