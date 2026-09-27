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

type ChatRepository interface {
	Create(ctx context.Context, chat *Chat, memberIDs []uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (*Chat, error)
	GetUserChats(ctx context.Context, userID uuid.UUID) ([]*Chat, error)
	IsMember(ctx context.Context, chatID, userID uuid.UUID) (bool, error)
}
