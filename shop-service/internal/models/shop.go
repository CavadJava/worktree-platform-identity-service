package models

import "time"

type Shop struct {
	ID          string    `json:"id"`
	OwnerID     string    `json:"owner_id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Temporary   bool      `json:"temporary"`
	ShopSeq     int64     `json:"shop_seq"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
