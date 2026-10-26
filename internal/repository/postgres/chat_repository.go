package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

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

func (r *ChatRepository) GetUserChats(ctx context.Context, userID uuid.UUID) ([]*domain.ChatPreview, error) {
	query := `
		SELECT 
			chats.id, 
			COALESCE(chats.title, direct_partner_users.username) AS title, 
			chats.type, 
			chats.created_at, 
			chats.updated_at,
			latest_messages.content          AS last_message_content,
			latest_message_authors.username  AS last_message_sender,
			latest_messages.created_at       AS last_message_time,
			COALESCE(unread_stats.count, 0)  AS unread_count,
			direct_partner_memberships.user_id AS direct_user_id
		FROM chats
		JOIN chat_members AS current_user_memberships 
		  ON chats.id = current_user_memberships.chat_id 
		 AND current_user_memberships.user_id = $1
		LEFT JOIN chat_members AS direct_partner_memberships 
		  ON chats.id = direct_partner_memberships.chat_id 
		 AND direct_partner_memberships.user_id != $1 
		 AND chats.type = 'direct'
		LEFT JOIN users AS direct_partner_users 
		  ON direct_partner_memberships.user_id = direct_partner_users.id
		LEFT JOIN LATERAL (
			SELECT messages.content, messages.sender_id, messages.created_at
			FROM messages
			WHERE messages.chat_id = chats.id
			ORDER BY messages.created_at DESC
			LIMIT 1
		) AS latest_messages ON true
		LEFT JOIN users AS latest_message_authors 
		  ON latest_messages.sender_id = latest_message_authors.id
		LEFT JOIN LATERAL (
			SELECT COUNT(*) AS count
			FROM messages
			WHERE messages.chat_id = chats.id
			  AND messages.created_at > current_user_memberships.last_read_at
			  AND (messages.sender_id IS NULL OR messages.sender_id != $1)
		) AS unread_stats ON true
		ORDER BY COALESCE(latest_messages.created_at, chats.updated_at) DESC;
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user chats: %w", err)
	}
	defer rows.Close()

	var previews []*domain.ChatPreview

	for rows.Next() {
		var preview domain.ChatPreview
		var (
			lastContent *string
			lastSender  *string
			lastTime    *time.Time
		)

		err := rows.Scan(
			&preview.ID,
			&preview.Title,
			&preview.Type,
			&preview.CreatedAt,
			&preview.UpdatedAt,
			&lastContent,
			&lastSender,
			&lastTime,
			&preview.UnreadCount,
			&preview.DirectUserID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan chat preview: %w", err)
		}

		if lastContent != nil {
			sender := ""
			if lastSender != nil {
				sender = *lastSender
			}
			t := time.Time{}
			if lastTime != nil {
				t = *lastTime
			}
			preview.LastMessage = &domain.LastMessagePreview{
				Content:        *lastContent,
				SenderUsername: sender,
				CreatedAt:      t,
			}
		}

		previews = append(previews, &preview)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during rows iteration: %w", err)
	}

	return previews, nil
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

func (r *ChatRepository) GetDirectChatBetweenUsers(ctx context.Context, user1ID, user2ID uuid.UUID) (*domain.Chat, error) {
	query := `
		SELECT c.id, c.title, c.type, c.created_at, c.updated_at
		FROM chats c
		JOIN chat_members cm1 ON c.id = cm1.chat_id AND cm1.user_id = $1
		JOIN chat_members cm2 ON c.id = cm2.chat_id AND cm2.user_id = $2
		WHERE c.type = 'direct'
		LIMIT 1;
	`

	var chat domain.Chat

	err := r.pool.QueryRow(ctx, query, user1ID, user2ID).Scan(&chat.ID, &chat.Title, &chat.Type, &chat.CreatedAt, &chat.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrChatNotFound
		}
		return nil, fmt.Errorf("failed to get chat: %w", err)
	}
	return &chat, nil
}

func (r *ChatRepository) GetChatMemberIDs(ctx context.Context, chatID uuid.UUID) ([]uuid.UUID, error) {
	query := `SELECT user_id FROM chat_members WHERE chat_id = $1`
	rows, err := r.pool.Query(ctx, query, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat members: %w", err)
	}
	defer rows.Close()
	var memberIDs []uuid.UUID
	for rows.Next() {
		var uid uuid.UUID
		if err := rows.Scan(&uid); err != nil {
			return nil, err
		}
		memberIDs = append(memberIDs, uid)
	}
	return memberIDs, rows.Err()
}

func (r *ChatRepository) MarkAsRead(ctx context.Context, chatID, userID uuid.UUID) error {
	query := `
		UPDATE chat_members 
		SET last_read_at = NOW() 
		WHERE chat_id = $1 AND user_id = $2
	`
	_, err := r.pool.Exec(ctx, query, chatID, userID)
	if err != nil {
		return fmt.Errorf("failed to mark chat as read: %w", err)
	}
	return nil
}
