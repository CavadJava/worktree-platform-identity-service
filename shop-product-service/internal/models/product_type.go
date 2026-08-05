package models

import "time"

// ProductType is a shop-defined category (növ) for organizing its products —
// e.g. "Ölçü" (Size). ProductSubtype is a sub-field within that category —
// e.g. "En" (Width), "Uzunluq" (Length).
type ProductType struct {
	ID        string    `json:"id"`
	ShopID    string    `json:"shop_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ProductSubtype struct {
	ID            string    `json:"id"`
	ProductTypeID string    `json:"product_type_id"`
	Name          string    `json:"name"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
