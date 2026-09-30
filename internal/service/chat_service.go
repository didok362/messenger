package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

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
	if strings.TrimSpace(content) == "" {
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

func (s *ChatService) CreateGroupChat(ctx context.Context, title string, creatorID uuid.UUID, memberIDs []uuid.UUID) (*domain.Chat, error) {
	if strings.TrimSpace(title) == "" {
		return nil, errors.New("group chat title cannot be empty")
	}

	allMembers := append(memberIDs, creatorID)

	chat := &domain.Chat{
		Title: &title,
		Type:  "group",
	}

	if err := s.chatRepo.Create(ctx, chat, allMembers); err != nil {
		return nil, fmt.Errorf("failed to create group chat: %w", err)
	}

	return chat, nil
}

func (s *ChatService) GetUserChats(ctx context.Context, userID uuid.UUID) ([]*domain.Chat, error) {
	return s.chatRepo.GetUserChats(ctx, userID)
}

func (s *ChatService) GetChatMessages(ctx context.Context, chatID, userID uuid.UUID, limit, offset int) ([]*domain.Message, error) {
	isMember, err := s.chatRepo.IsMember(ctx, chatID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check membership: %w", err)
	}
	if !isMember {
		return nil, domain.ErrForbidden
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return s.msgRepo.GetByChatID(ctx, chatID, limit, offset)
}
