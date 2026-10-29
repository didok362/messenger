package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/didok362/messenger/internal/domain"
	"github.com/didok362/messenger/internal/events"
	"github.com/google/uuid"
)

type MessageEventPublisher interface {
	PublishMessageCreated(ctx context.Context, event *events.MessageCreatedEvent) error
}

type ChatService struct {
	chatRepo domain.ChatRepository
	msgRepo  domain.MessageRepository
	userRepo domain.UserRepository
	hub      ChatHub
	producer MessageEventPublisher
}

type ChatHub interface {
	IsUserOnline(userID uuid.UUID) bool
	Broadcast(recipientIDs []uuid.UUID, payload any)
}

func NewChatService(chatRepo domain.ChatRepository, msgRepo domain.MessageRepository, userRepo domain.UserRepository, hub ChatHub, producer ...MessageEventPublisher) *ChatService {
	s := &ChatService{
		chatRepo: chatRepo,
		msgRepo:  msgRepo,
		userRepo: userRepo,
		hub:      hub,
	}
	if len(producer) > 0 {
		s.producer = producer[0]
	}
	return s
}

func (s *ChatService) GetUserChats(ctx context.Context, userID uuid.UUID) ([]*domain.ChatPreview, error) {
	chats, err := s.chatRepo.GetUserChats(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("Failed to get chats: %w", err)
	}
	if s.hub != nil {
		for _, chat := range chats {
			if chat.Type == "direct" && chat.DirectUserID != nil {
				chat.IsOnline = s.hub.IsUserOnline(*chat.DirectUserID)
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
		return nil, errors.New("Failed to fetch user")
	}
	memberIDs, err := s.chatRepo.GetChatMemberIDs(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to get members: %w", err)
	}
	now := time.Now().UTC()
	msg := domain.Message{
		ID:             uuid.New(), // Генерируем UUID прямо в Go
		ChatID:         chatID,
		SenderID:       &senderID,
		Content:        content,
		SenderUsername: user.Username,
		CreatedAt:      now,
	}
	// Если Kafka подключена — отправляем событие в стрим и сразу возвращаем ответ!
	if s.producer != nil {
		event := events.MessageCreatedEvent{
			ID:             msg.ID,
			ChatID:         msg.ChatID,
			SenderID:       msg.SenderID,
			SenderUsername: msg.SenderUsername,
			Content:        msg.Content,
			RecipientIDs:   memberIDs,
			CreatedAt:      msg.CreatedAt,
		}
		if err := s.producer.PublishMessageCreated(ctx, &event); err != nil {
			return nil, fmt.Errorf("failed to publish message event: %w", err)
		}
		return &msg, nil
	}
	// Фоллбэк (если Kafka отключена): старая синхронная запись в БД
	err = s.msgRepo.Create(ctx, &msg)
	if err != nil {
		return nil, errors.New("failed to create message")
	}
	if s.hub != nil {
		s.hub.Broadcast(memberIDs, &msg)
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
	if err := s.chatRepo.MarkAsRead(ctx, chatID, userID); err != nil {
		return err
	}
	// Уведомляем участников, что сообщения прочитаны
	if s.hub != nil {
		if memberIDs, err := s.chatRepo.GetChatMemberIDs(ctx, chatID); err == nil {
			s.hub.Broadcast(memberIDs, map[string]any{
				"type":      "messages_read",
				"chat_id":   chatID,
				"reader_id": userID,
			})
		}
	}
	return nil
}
