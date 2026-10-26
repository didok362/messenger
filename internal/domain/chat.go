package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Chat struct {
	ID        uuid.UUID `json:"id"`
	Title     *string   `json:"title"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type LastMessagePreview struct {
	Content        string    `json:"content"`
	SenderUsername string    `json:"sender_username"`
	CreatedAt      time.Time `json:"created_at"`
}

type ChatPreview struct {
	ID           uuid.UUID           `json:"id"`
	Title        *string             `json:"title"`
	Type         string              `json:"type"`
	LastMessage  *LastMessagePreview `json:"last_message,omitempty"`
	UnreadCount  int                 `json:"unread_count"`
	CreatedAt    time.Time           `json:"created_at"`
	UpdatedAt    time.Time           `json:"updated_at"`
	DirectUserID *uuid.UUID          `json:"direct_user_id,omitempty"`
	IsOnline     bool                `json:"is_online"`
}

type ChatRepository interface {
	Create(ctx context.Context, chat *Chat, memberIDs []uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (*Chat, error)
	GetUserChats(ctx context.Context, userID uuid.UUID) ([]*ChatPreview, error)
	IsMember(ctx context.Context, chatID, userID uuid.UUID) (bool, error)
	GetDirectChatBetweenUsers(ctx context.Context, user1ID, user2ID uuid.UUID) (*Chat, error)
	GetChatMemberIDs(ctx context.Context, chatID uuid.UUID) ([]uuid.UUID, error)
	MarkAsRead(ctx context.Context, chatID, userID uuid.UUID) error
}
