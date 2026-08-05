package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"shop-chat-service/internal/models"
	"shop-chat-service/internal/repository"
	"shop-chat-service/internal/roles"
)

var (
	ErrConversationNotFound = repository.ErrConversationNotFound
	ErrForbidden            = errors.New("forbidden")
	ErrBodyRequired         = errors.New("body is required")
	ErrShopIDRequired       = errors.New("shop_id is required")
)

// Identity is the calling user's claims, as returned by authorization-service.
type Identity struct {
	UserID        string
	Role          string
	ShopID        *string
	ShopRoleLevel int
}

func (i Identity) isSystemAdmin() bool {
	return i.Role == roles.RoleAdministrator
}

func (i Identity) isShopStaffOf(shopID string) bool {
	return i.ShopRoleLevel >= roles.ShopLevelChat && i.ShopID != nil && *i.ShopID == shopID
}

type ChatService struct {
	conversations *repository.ConversationRepository
	messages      *repository.MessageRepository
}

func NewChatService(conversations *repository.ConversationRepository, messages *repository.MessageRepository) *ChatService {
	return &ChatService{conversations: conversations, messages: messages}
}

// StartConversation gets-or-creates the single thread between the caller
// (as customer) and a shop — repeated calls with the same shop just return
// the existing thread.
func (s *ChatService) StartConversation(ctx context.Context, userID, shopID string) (*models.Conversation, error) {
	if shopID == "" {
		return nil, ErrShopIDRequired
	}

	existing, err := s.conversations.FindByShopAndUser(ctx, shopID, userID)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, repository.ErrConversationNotFound) {
		return nil, err
	}

	now := time.Now().UTC()
	conv := &models.Conversation{
		ID:        uuid.NewString(),
		ShopID:    shopID,
		UserID:    userID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.conversations.Create(ctx, conv); err != nil {
		return nil, err
	}
	return conv, nil
}

func (s *ChatService) ListMyConversations(ctx context.Context, userID string) ([]*models.Conversation, error) {
	return s.conversations.ListByUser(ctx, userID)
}

// ListShopConversations returns a shop's conversations — the caller must be
// that shop's chat(1)+ staff, or the system administrator.
func (s *ChatService) ListShopConversations(ctx context.Context, caller Identity, shopID string) ([]*models.Conversation, error) {
	if !caller.isSystemAdmin() && !caller.isShopStaffOf(shopID) {
		return nil, ErrForbidden
	}
	return s.conversations.ListByShop(ctx, shopID)
}

func (s *ChatService) ListMessages(ctx context.Context, caller Identity, conversationID string) ([]*models.Message, error) {
	conv, err := s.conversations.FindByID(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if !s.canAccess(caller, conv) {
		return nil, ErrForbidden
	}
	return s.messages.ListByConversation(ctx, conversationID)
}

func (s *ChatService) SendMessage(ctx context.Context, caller Identity, conversationID, body string) (*models.Message, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, ErrBodyRequired
	}

	conv, err := s.conversations.FindByID(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if !s.canAccess(caller, conv) {
		return nil, ErrForbidden
	}

	senderRole := models.SenderRoleUser
	if caller.UserID != conv.UserID {
		senderRole = models.SenderRoleShop
	}

	msg := &models.Message{
		ID:             uuid.NewString(),
		ConversationID: conversationID,
		SenderID:       caller.UserID,
		SenderRole:     senderRole,
		Body:           body,
		CreatedAt:      time.Now().UTC(),
	}
	if err := s.messages.Create(ctx, msg); err != nil {
		return nil, err
	}
	_ = s.conversations.Touch(ctx, conversationID)
	return msg, nil
}

// canAccess: the customer who owns the conversation, that shop's chat(1)+
// staff, or the system administrator.
func (s *ChatService) canAccess(caller Identity, conv *models.Conversation) bool {
	if caller.UserID == conv.UserID {
		return true
	}
	return caller.isSystemAdmin() || caller.isShopStaffOf(conv.ShopID)
}
