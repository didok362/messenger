package http

import (
	"encoding/json"
	"net/http"

	"github.com/didok362/messenger/internal/domain"
	"github.com/didok362/messenger/internal/service"
	"github.com/google/uuid"
)

type ChatHandler struct {
	ChatService *service.ChatService
}

type createDirectChatRequest struct {
	TargetUserID uuid.UUID `json:"target_user_id"`
}

func (h *ChatHandler) CreateDirect(w http.ResponseWriter, r *http.Request) {
	var req createDirectChatRequest
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "failed to decode request", http.StatusBadRequest)
		return
	}
	chat, err := h.ChatService.CreateDirectChat(r.Context(), userID, req.TargetUserID)
	if err != nil {
		http.Error(w, "faieled to create chat", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]*domain.Chat{
		"chat": chat,
	})
}

func NewChatHandler(chatService *service.ChatService) *ChatHandler {
	return &ChatHandler{ChatService: chatService}
}

func (h *ChatHandler) GetUserChats(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	chats, err := h.ChatService.GetUserChats(r.Context(), userID)
	if err != nil {
		http.Error(w, "failed to get chats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"chats": chats,
	})
}
