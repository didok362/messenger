package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/didok362/messenger/internal/domain"
	"github.com/didok362/messenger/internal/service"
	"github.com/google/uuid"
)

// mockChatRepo эмулирует репозиторий чатов
type mockChatRepo struct {
	createFn                    func(ctx context.Context, chat *domain.Chat, memberIDs []uuid.UUID) error
	isMemberFn                  func(ctx context.Context, chatID, userID uuid.UUID) (bool, error)
	getDirectChatBetweenUsersFn func(ctx context.Context, user1ID, user2ID uuid.UUID) (*domain.Chat, error)
	getChatMemberIDsFn          func(ctx context.Context, chatID uuid.UUID) ([]uuid.UUID, error)
	markAsReadFn                func(ctx context.Context, chatID, userID uuid.UUID) error
}

func (m *mockChatRepo) Create(ctx context.Context, chat *domain.Chat, memberIDs []uuid.UUID) error {
	if m.createFn != nil {
		return m.createFn(ctx, chat, memberIDs)
	}
	return nil
}
func (m *mockChatRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Chat, error) {
	return nil, nil
}
func (m *mockChatRepo) GetUserChats(ctx context.Context, userID uuid.UUID) ([]*domain.ChatPreview, error) {
	return nil, nil
}
func (m *mockChatRepo) IsMember(ctx context.Context, chatID, userID uuid.UUID) (bool, error) {
	if m.isMemberFn != nil {
		return m.isMemberFn(ctx, chatID, userID)
	}
	return true, nil
}
func (m *mockChatRepo) GetDirectChatBetweenUsers(ctx context.Context, user1ID, user2ID uuid.UUID) (*domain.Chat, error) {
	if m.getDirectChatBetweenUsersFn != nil {
		return m.getDirectChatBetweenUsersFn(ctx, user1ID, user2ID)
	}
	return nil, domain.ErrChatNotFound
}
func (m *mockChatRepo) GetChatMemberIDs(ctx context.Context, chatID uuid.UUID) ([]uuid.UUID, error) {
	if m.getChatMemberIDsFn != nil {
		return m.getChatMemberIDsFn(ctx, chatID)
	}
	return []uuid.UUID{}, nil
}
func (m *mockChatRepo) MarkAsRead(ctx context.Context, chatID, userID uuid.UUID) error {
	if m.markAsReadFn != nil {
		return m.markAsReadFn(ctx, chatID, userID)
	}
	return nil
}

// mockMsgRepo эмулирует репозиторий сообщений
type mockMsgRepo struct {
	createFn func(ctx context.Context, msg *domain.Message) error
}

func (m *mockMsgRepo) Create(ctx context.Context, msg *domain.Message) error {
	if m.createFn != nil {
		return m.createFn(ctx, msg)
	}
	return nil
}
func (m *mockMsgRepo) GetByChatID(ctx context.Context, chatID uuid.UUID, limit, offset int) ([]*domain.Message, error) {
	return nil, nil
}

// mockHub эмулирует WebSocket Hub и проверяет факт рассылки
type mockHub struct {
	broadcastCalled bool
	recipients      []uuid.UUID
	payload         any
}

func (m *mockHub) IsUserOnline(userID uuid.UUID) bool {
	return true
}

func (m *mockHub) Broadcast(recipientIDs []uuid.UUID, payload any) {
	m.broadcastCalled = true
	m.recipients = recipientIDs
	m.payload = payload
}

func TestChatService_CreateDirectChat(t *testing.T) {
	ctx := context.Background()
	user1 := uuid.New()
	user2 := uuid.New()

	t.Run("cannot_create_chat_with_yourself", func(t *testing.T) {
		chatService := service.NewChatService(&mockChatRepo{}, &mockMsgRepo{}, &mockUserRepo{}, &mockHub{})
		_, err := chatService.CreateDirectChat(ctx, user1, user1)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns_existing_chat", func(t *testing.T) {
		existingChat := &domain.Chat{ID: uuid.New(), Type: "direct"}
		chatRepo := &mockChatRepo{
			getDirectChatBetweenUsersFn: func(ctx context.Context, u1, u2 uuid.UUID) (*domain.Chat, error) {
				return existingChat, nil
			},
		}

		chatService := service.NewChatService(chatRepo, &mockMsgRepo{}, &mockUserRepo{}, &mockHub{})
		chat, err := chatService.CreateDirectChat(ctx, user1, user2)

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if chat.ID != existingChat.ID {
			t.Errorf("expected chat ID %s, got %s", existingChat.ID, chat.ID)
		}
	})
}

func TestChatService_SendMessage(t *testing.T) {
	ctx := context.Background()
	chatID := uuid.New()
	senderID := uuid.New()
	recipientID := uuid.New()

	t.Run("empty_content_error", func(t *testing.T) {
		chatService := service.NewChatService(&mockChatRepo{}, &mockMsgRepo{}, &mockUserRepo{}, &mockHub{})
		_, err := chatService.SendMessage(ctx, "   ", chatID, senderID)
		if err == nil {
			t.Fatal("expected error for empty message, got nil")
		}
	})

	t.Run("not_a_member_error", func(t *testing.T) {
		chatRepo := &mockChatRepo{
			isMemberFn: func(ctx context.Context, cID, uID uuid.UUID) (bool, error) {
				return false, nil // не участник чата
			},
		}

		chatService := service.NewChatService(chatRepo, &mockMsgRepo{}, &mockUserRepo{}, &mockHub{})
		_, err := chatService.SendMessage(ctx, "hello", chatID, senderID)
		if err == nil {
			t.Fatal("expected error for non-member, got nil")
		}
	})

	t.Run("success_and_broadcast", func(t *testing.T) {
		chatRepo := &mockChatRepo{
			isMemberFn: func(ctx context.Context, cID, uID uuid.UUID) (bool, error) {
				return true, nil
			},
			getChatMemberIDsFn: func(ctx context.Context, cID uuid.UUID) ([]uuid.UUID, error) {
				return []uuid.UUID{senderID, recipientID}, nil
			},
		}

		userRepo := &mockUserRepo{
			getByIDFn: func(ctx context.Context, id uuid.UUID) (*domain.User, error) {
				return &domain.User{ID: id, Username: "alex"}, nil
			},
		}

		msgRepo := &mockMsgRepo{
			createFn: func(ctx context.Context, msg *domain.Message) error {
				msg.ID = uuid.New()
				msg.CreatedAt = time.Now()
				return nil
			},
		}

		hub := &mockHub{}
		chatService := service.NewChatService(chatRepo, msgRepo, userRepo, hub)

		msg, err := chatService.SendMessage(ctx, "Привет, мир!", chatID, senderID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if msg.Content != "Привет, мир!" || msg.SenderUsername != "alex" {
			t.Errorf("unexpected message: %+v", msg)
		}

		// Проверяем, что хаб реально вызвал рассылку участникам
		if !hub.broadcastCalled {
			t.Error("expected hub.Broadcast to be called")
		}
		if len(hub.recipients) != 2 {
			t.Errorf("expected 2 recipients, got %d", len(hub.recipients))
		}
	})
}

func TestChatService_MarkChatAsRead(t *testing.T) {
	ctx := context.Background()
	chatID := uuid.New()
	userID := uuid.New()

	t.Run("success_and_broadcast_read_receipt", func(t *testing.T) {
		markAsReadCalled := false
		chatRepo := &mockChatRepo{
			isMemberFn: func(ctx context.Context, cID, uID uuid.UUID) (bool, error) {
				return true, nil
			},
			markAsReadFn: func(ctx context.Context, cID, uID uuid.UUID) error {
				markAsReadCalled = true
				return nil
			},
			getChatMemberIDsFn: func(ctx context.Context, cID uuid.UUID) ([]uuid.UUID, error) {
				return []uuid.UUID{userID}, nil
			},
		}

		hub := &mockHub{}
		chatService := service.NewChatService(chatRepo, &mockMsgRepo{}, &mockUserRepo{}, hub)

		err := chatService.MarkChatAsRead(ctx, chatID, userID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !markAsReadCalled {
			t.Error("expected MarkAsRead to be called in repo")
		}
		if !hub.broadcastCalled {
			t.Error("expected hub.Broadcast to be called for read receipt")
		}
	})

	t.Run("forbidden_for_non_member", func(t *testing.T) {
		chatRepo := &mockChatRepo{
			isMemberFn: func(ctx context.Context, cID, uID uuid.UUID) (bool, error) {
				return false, nil
			},
		}

		chatService := service.NewChatService(chatRepo, &mockMsgRepo{}, &mockUserRepo{}, &mockHub{})
		err := chatService.MarkChatAsRead(ctx, chatID, userID)

		if !errors.Is(err, domain.ErrForbidden) {
			t.Fatalf("expected ErrForbidden, got %v", err)
		}
	})
}

func TestChatService_CreateGroupChat(t *testing.T) {
	ctx := context.Background()
	creatorID := uuid.New()
	member1 := uuid.New()

	t.Run("empty_title_error", func(t *testing.T) {
		chatService := service.NewChatService(&mockChatRepo{}, &mockMsgRepo{}, &mockUserRepo{}, &mockHub{})
		_, err := chatService.CreateGroupChat(ctx, "   ", creatorID, []uuid.UUID{member1})
		if err == nil {
			t.Fatal("expected error for empty title, got nil")
		}
	})

	t.Run("success_with_deduplicated_members", func(t *testing.T) {
		var passedMembers []uuid.UUID
		chatRepo := &mockChatRepo{
			createFn: func(ctx context.Context, chat *domain.Chat, memberIDs []uuid.UUID) error {
				chat.ID = uuid.New()
				passedMembers = memberIDs
				return nil
			},
		}

		chatService := service.NewChatService(chatRepo, &mockMsgRepo{}, &mockUserRepo{}, &mockHub{})
		// Передаем дубликат создателя и участника:
		chat, err := chatService.CreateGroupChat(ctx, "Gopnik Party", creatorID, []uuid.UUID{creatorID, member1, member1})

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if *chat.Title != "Gopnik Party" || chat.Type != "group" {
			t.Errorf("unexpected chat: %+v", chat)
		}
		// Дубликаты должны удалиться: ровно 2 уникальных участника
		if len(passedMembers) != 2 {
			t.Errorf("expected 2 unique members, got %d", len(passedMembers))
		}
	})
}

func TestChatService_GetChatMessages(t *testing.T) {
	ctx := context.Background()
	chatID := uuid.New()
	userID := uuid.New()

	t.Run("forbidden_for_non_member", func(t *testing.T) {
		chatRepo := &mockChatRepo{
			isMemberFn: func(ctx context.Context, cID, uID uuid.UUID) (bool, error) {
				return false, nil
			},
		}
		chatService := service.NewChatService(chatRepo, &mockMsgRepo{}, &mockUserRepo{}, &mockHub{})
		_, err := chatService.GetChatMessages(ctx, chatID, userID, 10, 0)
		if !errors.Is(err, domain.ErrForbidden) {
			t.Fatalf("expected ErrForbidden, got %v", err)
		}
	})
}
