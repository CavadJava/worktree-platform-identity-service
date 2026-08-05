package models

import "time"

type User struct {
	ID            string    `json:"id"`
	Email         string    `json:"email"`
	PasswordHash  string    `json:"-"`
	FullName      string    `json:"full_name"`
	Phone         string    `json:"phone,omitempty"`
	Role          string    `json:"role"`
	ShopID        *string   `json:"shop_id,omitempty"`
	ShopRoleLevel int       `json:"shop_role_level"`
	UserSeq       int64     `json:"user_seq"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
