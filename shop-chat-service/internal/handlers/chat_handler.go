package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"shop-chat-service/internal/client"
	"shop-chat-service/internal/middleware"
	_ "shop-chat-service/internal/models" // referenced by swag annotations
	"shop-chat-service/internal/service"
)

type ChatHandler struct {
	svc *service.ChatService
}

func NewChatHandler(svc *service.ChatService) *ChatHandler {
	return &ChatHandler{svc: svc}
}

type startConversationRequest struct {
	ShopID string `json:"shop_id"`
}

type sendMessageRequest struct {
	Body string `json:"body"`
}

// StartConversation godoc
// @Summary      Start (or resume) a conversation with a shop
// @Description  İstənilən login olmuş istifadəçi çağıra bilər. Eyni shop_id ilə təkrar çağırış mövcud söhbəti qaytarır.
// @Tags         chat
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body startConversationRequest true "Start conversation payload"
// @Success      201 {object} models.Conversation
// @Failure      400 {object} map[string]string
// @Router       /conversations [post]
func (h *ChatHandler) StartConversation(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req startConversationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	conv, err := h.svc.StartConversation(r.Context(), identity.UserID, req.ShopID)
	if err != nil {
		if errors.Is(err, service.ErrShopIDRequired) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to start conversation")
		return
	}

	writeJSON(w, http.StatusCreated, conv)
}

// ListMyConversations godoc
// @Summary      List my conversations (as a customer)
// @Tags         chat
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} models.Conversation
// @Router       /conversations [get]
func (h *ChatHandler) ListMyConversations(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	conversations, err := h.svc.ListMyConversations(r.Context(), identity.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list conversations")
		return
	}
	writeJSON(w, http.StatusOK, conversations)
}

// ListShopConversations godoc
// @Summary      List a shop's conversations
// @Description  Sistem administratoru istənilən mağazanın, mağazanın öz chat(1)+ əməkdaşı isə yalnız öz mağazasının söhbətlərini görə bilər.
// @Tags         chat
// @Produce      json
// @Security     BearerAuth
// @Param        shop_id path string true "Shop ID"
// @Success      200 {array} models.Conversation
// @Failure      403 {object} map[string]string
// @Router       /shops/{shop_id}/conversations [get]
func (h *ChatHandler) ListShopConversations(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	shopID := chi.URLParam(r, "shop_id")
	conversations, err := h.svc.ListShopConversations(r.Context(), toIdentity(identity), shopID)
	if err != nil {
		if errors.Is(err, service.ErrForbidden) {
			writeError(w, http.StatusForbidden, "administrator, or this shop's chat(1)+ staff, required")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to list conversations")
		return
	}
	writeJSON(w, http.StatusOK, conversations)
}

// ListMessages godoc
// @Summary      List a conversation's messages
// @Tags         chat
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Conversation ID"
// @Success      200 {array} models.Message
// @Failure      403 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /conversations/{id}/messages [get]
func (h *ChatHandler) ListMessages(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	messages, err := h.svc.ListMessages(r.Context(), toIdentity(identity), id)
	if err != nil {
		writeMessageError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, messages)
}

// SendMessage godoc
// @Summary      Send a message in a conversation
// @Description  Söhbətin sahibi olan müştəri, ya da mağazanın chat(1)+ əməkdaşı (və ya administrator) yaza bilər.
// @Tags         chat
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Conversation ID"
// @Param        request body sendMessageRequest true "Message payload"
// @Success      201 {object} models.Message
// @Failure      400 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /conversations/{id}/messages [post]
func (h *ChatHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")

	var req sendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	msg, err := h.svc.SendMessage(r.Context(), toIdentity(identity), id, req.Body)
	if err != nil {
		if errors.Is(err, service.ErrBodyRequired) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeMessageError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, msg)
}

func writeMessageError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrConversationNotFound):
		writeError(w, http.StatusNotFound, "conversation not found")
	case errors.Is(err, service.ErrForbidden):
		writeError(w, http.StatusForbidden, "not a participant in this conversation")
	default:
		writeError(w, http.StatusInternalServerError, "failed to process request")
	}
}

func toIdentity(identity *client.Identity) service.Identity {
	return service.Identity{
		UserID:        identity.UserID,
		Role:          identity.Role,
		ShopID:        identity.ShopID,
		ShopRoleLevel: identity.ShopRoleLevel,
	}
}
