package ws

import (
	"context"
	"log"
	"sync"

	"github.com/didok362/messenger/internal/domain"
	"github.com/google/uuid"
)

type chatBroadcast struct {
	chatID  uuid.UUID
	payload any
}

type Hub struct {
	// Внешний ключ: UserID. Внутренний: сокет-сессии пользователя (ПК, телефон и т.д.)
	clients        map[uuid.UUID]map[*Client]bool
	broadcast      chan *domain.Message
	broadcastEvent chan *chatBroadcast
	register       chan *Client
	unregister     chan *Client
	chatRepo       domain.ChatRepository
	mu             sync.RWMutex
}

func NewHub(chatRepo domain.ChatRepository) *Hub {
	return &Hub{
		clients:        make(map[uuid.UUID]map[*Client]bool),
		broadcast:      make(chan *domain.Message),
		broadcastEvent: make(chan *chatBroadcast),
		register:       make(chan *Client),
		unregister:     make(chan *Client),
		chatRepo:       chatRepo,
	}
}

func (h *Hub) BroadcastToChat(chatID uuid.UUID, payload any) {
	h.broadcastEvent <- &chatBroadcast{
		chatID:  chatID,
		payload: payload,
	}
}

func (h *Hub) IsUserOnline(userID uuid.UUID) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[userID]) > 0
}

func (h *Hub) broadcastGlobal(payload any) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, userConns := range h.clients {
		for client := range userConns {
			select {
			case client.send <- payload: // если буфер свободен — отправляем мгновенно
			default: // если клиент завис и его буфер переполнен — просто пропускаем его, сервер не ждет!
			}
		}
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			wasOffline := len(h.clients[client.userID]) == 0
			userConns := h.clients[client.userID]
			if userConns == nil {
				userConns = make(map[*Client]bool)
				h.clients[client.userID] = userConns
			}
			userConns[client] = true
			h.mu.Unlock()
			if wasOffline {
				h.broadcastGlobal(map[string]any{
					"type":      "user_status",
					"user_id":   client.userID,
					"is_online": true,
				})
			}

		case client := <-h.unregister:
			h.mu.Lock()
			wentOffline := false
			if userConns, ok := h.clients[client.userID]; ok {
				if _, exists := userConns[client]; exists {
					delete(userConns, client)
					close(client.send)
					// Если у пользователя не осталось активных вкладок/девайсов — чистим юзера
					if len(userConns) == 0 {
						delete(h.clients, client.userID)
					}
				}
			}
			if len(h.clients[client.userID]) == 0 {
				wentOffline = true
			}
			h.mu.Unlock()
			if wentOffline {
				h.broadcastGlobal(map[string]any{
					"type":      "user_status",
					"user_id":   client.userID,
					"is_online": false,
				})
			}
		case message := <-h.broadcast:
			h.sendToChatMembers(message.ChatID, message)

		case event := <-h.broadcastEvent:
			h.sendToChatMembers(event.chatID, event.payload)
		}
	}
}

func (h *Hub) sendToChatMembers(chatID uuid.UUID, payload any) {
	memberIDs, err := h.chatRepo.GetChatMemberIDs(context.Background(), chatID)
	if err != nil {
		log.Printf("WS Hub: failed to get members for chat %s: %v", chatID, err)
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, memberID := range memberIDs {
		if userConns, ok := h.clients[memberID]; ok {
			for client := range userConns {
				select {
				case client.send <- payload:
				default:
				}
			}
		}
	}
}
