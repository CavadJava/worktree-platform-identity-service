package repository

import (
	"context"
	"database/sql"

	"shop-chat-service/internal/models"
)

type MessageRepository struct {
	db *sql.DB
}

func NewMessageRepository(db *sql.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

func (r *MessageRepository) Create(ctx context.Context, m *models.Message) error {
	const q = `
		INSERT INTO messages (id, conversation_id, sender_id, sender_role, body, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.ExecContext(ctx, q, m.ID, m.ConversationID, m.SenderID, m.SenderRole, m.Body, m.CreatedAt)
	return err
}

func (r *MessageRepository) ListByConversation(ctx context.Context, conversationID string) ([]*models.Message, error) {
	const q = `
		SELECT id, conversation_id, sender_id, sender_role, body, created_at
		FROM messages WHERE conversation_id = $1
		ORDER BY created_at ASC
	`
	rows, err := r.db.QueryContext(ctx, q, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := []*models.Message{}
	for rows.Next() {
		m := &models.Message{}
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.SenderRole, &m.Body, &m.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, rows.Err()
}
