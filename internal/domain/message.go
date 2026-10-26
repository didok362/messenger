package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Message struct {
	ID             uuid.UUID  `json:"id"`
	ChatID         uuid.UUID  `json:"chat_id"`
	SenderID       *uuid.UUID `json:"sender_id"`
	SenderUsername string     `json:"sender_username,omitempty"`
	Content        string     `json:"content"`
	CreatedAt      time.Time  `json:"created_at"`
	IsRead         bool       `json:"is_read"`
}

type MessageRepository interface {
	Create(ctx context.Context, msg *Message) error
	GetByChatID(ctx context.Context, chatID uuid.UUID, limit, offset int) ([]*Message, error)
}
