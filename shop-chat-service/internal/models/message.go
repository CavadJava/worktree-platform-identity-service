package models

import "time"

const (
	SenderRoleUser = "user"
	SenderRoleShop = "shop"
)

type Message struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversation_id"`
	SenderID       string    `json:"sender_id"`
	SenderRole     string    `json:"sender_role"`
	Body           string    `json:"body"`
	CreatedAt      time.Time `json:"created_at"`
}
