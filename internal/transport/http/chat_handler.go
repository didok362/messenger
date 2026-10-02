package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/didok362/messenger/internal/domain"
	"github.com/didok362/messenger/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type ChatHandler struct {
	ChatService *service.ChatService
}

type createDirectChatRequest struct {
	TargetUserID uuid.UUID `json:"target_user_id"`
}

type sendMessageRequest struct {
	Content string `json:"content"`
}

func NewChatHandler(chatService *service.ChatService) *ChatHandler {
	return &ChatHandler{ChatService: chatService}
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

func (h *ChatHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	var req sendMessageRequest
	chatIDStr := chi.URLParam(r, "id")
	chatID, err := uuid.Parse(chatIDStr)
	if err != nil {
		http.Error(w, "invalid chat id", http.StatusBadRequest)
		return
	}
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "failed to decode request", http.StatusBadRequest)
		return
	}
	msg, err := h.ChatService.SendMessage(r.Context(), req.Content, chatID, userID)
	if err != nil {
		http.Error(w, "faieled to send message", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]*domain.Message{
		"message": msg,
	})
}

func (h *ChatHandler) GetChatMessages(w http.ResponseWriter, r *http.Request) {
	chatIDStr := chi.URLParam(r, "id")
	chatID, err := uuid.Parse(chatIDStr)
	if err != nil {
		http.Error(w, "invalid chat id", http.StatusBadRequest)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	UserID, ok := UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	messages, err := h.ChatService.GetChatMessages(r.Context(), chatID, UserID, limit, offset)
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			http.Error(w, "failed to get chats", http.StatusForbidden)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"messages": messages,
	})
}
