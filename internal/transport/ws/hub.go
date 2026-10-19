package ws

import (
	"github.com/didok362/messenger/internal/domain"
)

type Hub struct {
	// Зарегистрированные клиенты (ключ — указатель на клиента, значение — bool)
	clients map[*Client]bool
	// Канал входящих сообщений от клиентов для рассылки
	broadcast chan *domain.Message
	// Канал для регистрации новых подключений
	register chan *Client
	// Канал для удаления отключившихся клиентов
	unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan *domain.Message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:

				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}
