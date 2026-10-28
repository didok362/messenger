package ws

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/didok362/messenger/internal/metrics"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const redisChannel = "messenger:broadcast"

type BroadcastPayload struct {
	RecipientIDs []uuid.UUID
	Payload      any
}

type Hub struct {
	// Внешний ключ: UserID. Внутренний: сокет-сессии пользователя (ПК, телефон и т.д.)
	clients    map[uuid.UUID]map[*Client]bool
	broadcast  chan BroadcastPayload
	register   chan *Client
	unregister chan *Client
	rdb        *redis.Client
	mu         sync.RWMutex
}

func NewHub(rdb *redis.Client) *Hub {
	h := &Hub{
		clients:    make(map[uuid.UUID]map[*Client]bool),
		broadcast:  make(chan BroadcastPayload, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		rdb:        rdb,
	}

	if rdb != nil {
		go h.subscribeRedis()
	}

	return h
}

func (h *Hub) Broadcast(recipientIDs []uuid.UUID, payload any) {
	if len(recipientIDs) == 0 {
		return
	}

	msg := BroadcastPayload{
		RecipientIDs: recipientIDs,
		Payload:      payload,
	}

	if h.rdb != nil {
		if data, err := json.Marshal(msg); err == nil {
			_ = h.rdb.Publish(context.Background(), redisChannel, data).Err()
			return
		}
	}

	h.broadcast <- msg
}

func (h *Hub) subscribeRedis() {
	pubsub := h.rdb.Subscribe(context.Background(), redisChannel)
	defer pubsub.Close()
	ch := pubsub.Channel()
	for msg := range ch {
		var payload BroadcastPayload
		if err := json.Unmarshal([]byte(msg.Payload), &payload); err == nil {
			h.broadcast <- payload
		}
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
			metrics.WebSocketActiveConnections.Inc()
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
			metrics.WebSocketActiveConnections.Dec()
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
		case msg := <-h.broadcast:
			start := time.Now()
			h.mu.RLock()
			for _, id := range msg.RecipientIDs {
				if userConns, ok := h.clients[id]; ok {
					for client := range userConns {
						select {
						case client.send <- msg.Payload:
						default:
						}
					}
				}
			}
			h.mu.RUnlock()
			metrics.WebSocketBroadcastDuration.Observe(time.Since(start).Seconds())
		}
	}
}
