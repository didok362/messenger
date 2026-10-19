package ws

import (
	"context"

	"github.com/didok362/messenger/internal/domain"
	"github.com/didok362/messenger/internal/service"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Client struct {
	hub         *Hub
	conn        *websocket.Conn
	send        chan *domain.Message
	userID      uuid.UUID
	chatService *service.ChatService
}

type incomingWSMessage struct {
	ChatID  uuid.UUID `json:"chat_id"`
	Content string    `json:"content"`
}

func (c *Client) writePump() {
	defer func() {
		_ = c.conn.Close()
	}()
	for message := range c.send {
		err := c.conn.WriteJSON(message)
		if err != nil {
			return
		}
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		_ = c.conn.Close()
	}()
	for {
		var msg incomingWSMessage
		if err := c.conn.ReadJSON(&msg); err != nil {
			break
		}
		savedMsg, err := c.chatService.SendMessage(context.Background(), msg.Content, msg.ChatID, c.userID)
		if err != nil {
			continue
		}
		c.hub.broadcast <- savedMsg
	}

}
