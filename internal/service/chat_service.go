package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/didok362/messenger/internal/domain"
	"github.com/google/uuid"
)

type ChatService struct {
	chatRepo domain.ChatRepository
	msgRepo  domain.MessageRepository
}

func NewChatService(chatRepo domain.ChatRepository, msgRepo domain.MessageRepository) *ChatService {
	return &ChatService{
		chatRepo: chatRepo,
		msgRepo:  msgRepo,
	}
}

func (s *ChatService) CreateDirectChat(ctx context.Context, user1ID, user2ID uuid.UUID) (*domain.Chat, error) {
	if user1ID == user2ID {
		return nil, errors.New("cannot create chat with yourself")
	}
	chat := &domain.Chat{Type: "direct", Title: nil}
	if err := s.chatRepo.Create(ctx, chat, []uuid.UUID{user1ID, user2ID}); err != nil {
		return nil, fmt.Errorf("faield to create chat: %w", err)
	}

	return chat, nil
}

func (s *ChatService) SendMessage(ctx context.Context, content string, chatID uuid.UUID, senderID uuid.UUID) (*domain.Message, error) {
	if content == "" {
		return nil, errors.New("cannot send empty msg")
	}
	isMember, err := s.chatRepo.IsMember(ctx, chatID, senderID)
	if err != nil {
		return nil, errors.New("failed to check membership")
	}
	if !isMember {
		return nil, errors.New("You are not a member")
	}

	msg := domain.Message{
		ChatID:   chatID,
		SenderID: &senderID,
		Content:  content,
	}

	err = s.msgRepo.Create(ctx, &msg)
	if err != nil {
		return nil, errors.New("failed to create user")
	}
	return &msg, nil
}
