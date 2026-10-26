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
	userRepo domain.UserRepository
	presence PresenceChecker
}

type PresenceChecker interface {
	IsUserOnline(userID uuid.UUID) bool
}

func NewChatService(chatRepo domain.ChatRepository, msgRepo domain.MessageRepository, userRepo domain.UserRepository, presence PresenceChecker) *ChatService {
	return &ChatService{
		chatRepo: chatRepo,
		msgRepo:  msgRepo,
		userRepo: userRepo,
		presence: presence,
	}
}

func (s *ChatService) GetUserChats(ctx context.Context, userID uuid.UUID) ([]*domain.ChatPreview, error) {
	chats, err := s.chatRepo.GetUserChats(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("Failed to get chats: %w", err)
	}
	if s.presence != nil {
		for _, chat := range chats {
			if chat.Type == "direct" && chat.DirectUserID != nil {
				chat.IsOnline = s.presence.IsUserOnline(*chat.DirectUserID)
			}
		}
	}
	return chats, nil
}

func (s *ChatService) CreateDirectChat(ctx context.Context, user1ID, user2ID uuid.UUID) (*domain.Chat, error) {
	if user1ID == user2ID {
		return nil, errors.New("cannot create chat with yourself")
	}

	chat, err := s.chatRepo.GetDirectChatBetweenUsers(ctx, user1ID, user2ID)

	if err == nil {
		return chat, nil
	}

	if !errors.Is(err, domain.ErrChatNotFound) {
		return nil, fmt.Errorf("failed to check existing chat: %w", err)
	}

	chat = &domain.Chat{Type: "direct", Title: nil}
	if err := s.chatRepo.Create(ctx, chat, []uuid.UUID{user1ID, user2ID}); err != nil {
		return nil, fmt.Errorf("failed to create chat: %w", err)
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

	user, err := s.userRepo.GetByID(ctx, senderID)
	if err != nil {
		return nil, errors.New("Failed to fecth user")
	}

	msg := domain.Message{
		ChatID:         chatID,
		SenderID:       &senderID,
		Content:        content,
		SenderUsername: user.Username,
	}

	err = s.msgRepo.Create(ctx, &msg)
	if err != nil {
		return nil, errors.New("failed to create message")
	}
	return &msg, nil
}

func (s *ChatService) CreateGroupChat(ctx context.Context, title string, creatorID uuid.UUID, memberIDs []uuid.UUID) (*domain.Chat, error) {
	if strings.TrimSpace(title) == "" {
		return nil, errors.New("group chat title cannot be empty")
	}

	membersMap := make(map[uuid.UUID]struct{})

	membersMap[creatorID] = struct{}{}

	for _, id := range memberIDs {
		membersMap[id] = struct{}{}
	}

	allMembers := make([]uuid.UUID, 0, len(membersMap))
	for id := range membersMap {
		allMembers = append(allMembers, id)
	}

	chat := &domain.Chat{
		Title: &title,
		Type:  "group",
	}

	if err := s.chatRepo.Create(ctx, chat, allMembers); err != nil {
		return nil, fmt.Errorf("failed to create group chat: %w", err)
	}

	return chat, nil
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

func (s *ChatService) MarkChatAsRead(ctx context.Context, chatID, userID uuid.UUID) error {
	isMember, err := s.chatRepo.IsMember(ctx, chatID, userID)
	if err != nil {
		return fmt.Errorf("failed to check membership: %w", err)
	}
	if !isMember {
		return domain.ErrForbidden
	}
	return s.chatRepo.MarkAsRead(ctx, chatID, userID)
}
