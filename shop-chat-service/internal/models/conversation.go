package models

import "time"

type Conversation struct {
	ID        string    `json:"id"`
	ShopID    string    `json:"shop_id"`
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
