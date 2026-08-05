package repository

import (
	"context"
	"database/sql"
	"errors"

	"shop-chat-service/internal/models"
)

var ErrConversationNotFound = errors.New("conversation not found")

type ConversationRepository struct {
	db *sql.DB
}

func NewConversationRepository(db *sql.DB) *ConversationRepository {
	return &ConversationRepository{db: db}
}

func (r *ConversationRepository) Create(ctx context.Context, c *models.Conversation) error {
	const q = `
		INSERT INTO conversations (id, shop_id, user_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.ExecContext(ctx, q, c.ID, c.ShopID, c.UserID, c.CreatedAt, c.UpdatedAt)
	return err
}

func (r *ConversationRepository) FindByID(ctx context.Context, id string) (*models.Conversation, error) {
	const q = `
		SELECT id, shop_id, user_id, created_at, updated_at
		FROM conversations WHERE id = $1
	`
	c := &models.Conversation{}
	err := r.db.QueryRowContext(ctx, q, id).Scan(&c.ID, &c.ShopID, &c.UserID, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrConversationNotFound
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

// FindByShopAndUser looks up the single conversation between one shop and
// one customer — used by GetOrCreate to enforce the one-thread-per-pair rule.
func (r *ConversationRepository) FindByShopAndUser(ctx context.Context, shopID, userID string) (*models.Conversation, error) {
	const q = `
		SELECT id, shop_id, user_id, created_at, updated_at
		FROM conversations WHERE shop_id = $1 AND user_id = $2
	`
	c := &models.Conversation{}
	err := r.db.QueryRowContext(ctx, q, shopID, userID).Scan(&c.ID, &c.ShopID, &c.UserID, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrConversationNotFound
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (r *ConversationRepository) ListByUser(ctx context.Context, userID string) ([]*models.Conversation, error) {
	const q = `
		SELECT id, shop_id, user_id, created_at, updated_at
		FROM conversations WHERE user_id = $1
		ORDER BY updated_at DESC
	`
	return r.list(ctx, q, userID)
}

func (r *ConversationRepository) ListByShop(ctx context.Context, shopID string) ([]*models.Conversation, error) {
	const q = `
		SELECT id, shop_id, user_id, created_at, updated_at
		FROM conversations WHERE shop_id = $1
		ORDER BY updated_at DESC
	`
	return r.list(ctx, q, shopID)
}

func (r *ConversationRepository) list(ctx context.Context, q, arg string) ([]*models.Conversation, error) {
	rows, err := r.db.QueryContext(ctx, q, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	conversations := []*models.Conversation{}
	for rows.Next() {
		c := &models.Conversation{}
		if err := rows.Scan(&c.ID, &c.ShopID, &c.UserID, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		conversations = append(conversations, c)
	}
	return conversations, rows.Err()
}

func (r *ConversationRepository) Touch(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE conversations SET updated_at = now() WHERE id = $1`, id)
	return err
}
