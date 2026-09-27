package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/didok362/messenger/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ChatRepository struct {
	pool *pgxpool.Pool
}

func NewChatRepository(pool *pgxpool.Pool) *ChatRepository {
	return &ChatRepository{pool: pool}
}

func (r *ChatRepository) Create(ctx context.Context, chat *domain.Chat, memberIDs []uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to createa pool: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queryRow := `
		INSERT INTO chats (title, type)
		VALUES ($1, $2)
		RETURNING id, created_at, updated_at
	`
	err = tx.QueryRow(ctx, queryRow, chat.Title, chat.Type).Scan(&chat.ID, &chat.CreatedAt, &chat.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to createa chat: %w", err)
	}
	queryMember := `
			INSERT INTO chat_members (chat_id, user_id) VALUES ($1, $2)
		`
	for _, memberID := range memberIDs {

		if _, err := tx.Exec(ctx, queryMember, chat.ID, memberID); err != nil {
			return fmt.Errorf("failed to insert chat member: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to add members: %w", err)
	}
	return nil
}

func (r *ChatRepository) GetUserChats(ctx context.Context, userID uuid.UUID) ([]*domain.Chat, error) {
	query := `
		SELECT 
			chats.id, 
			chats.title, 
			chats.type, 
			chats.created_at, 
			chats.updated_at
		FROM chats
		JOIN chat_members ON chats.id = chat_members.chat_id
		WHERE chat_members.user_id = $1
		ORDER BY chats.updated_at DESC
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user chats: %w", err)
	}
	defer rows.Close()
	var chats []*domain.Chat

	for rows.Next() {
		var chat domain.Chat
		err := rows.Scan(
			&chat.ID,
			&chat.Title,
			&chat.Type,
			&chat.CreatedAt,
			&chat.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan chat row: %w", err)
		}
		chats = append(chats, &chat)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during rows iteration: %w", err)
	}

	return chats, nil
}

func (r *ChatRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Chat, error) {

	var chat domain.Chat
	query := `
		SELECT id, title, type, created_at, updated_at
		FROM chats
		WHERE id = $1
	`
	err := r.pool.QueryRow(ctx, query, id).Scan(&chat.ID, &chat.Title, &chat.Type, &chat.CreatedAt, &chat.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrChatNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &chat, nil

}

func (r *ChatRepository) IsMember(ctx context.Context, chatID uuid.UUID, userID uuid.UUID) (bool, error) {
	query := ` 
	SELECT EXISTS(
    SELECT 1 FROM chat_members 
    WHERE chat_id = $1 AND user_id = $2)
	`
	var exists bool
	err := r.pool.QueryRow(ctx, query, chatID, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check membership: %w", err)
	}
	return exists, nil
}
