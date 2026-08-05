package models

import "time"

type Address struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Title       string    `json:"title"`
	FullAddress string    `json:"full_address"`
	City        string    `json:"city,omitempty"`
	Phone       string    `json:"phone,omitempty"`
	IsDefault   bool      `json:"is_default"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
