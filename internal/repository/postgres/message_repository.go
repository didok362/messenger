package postgres

import (
	"context"
	"fmt"

	"github.com/didok362/messenger/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MessageRepository struct {
	pool *pgxpool.Pool
}

func NewMessageRepository(pool *pgxpool.Pool) *MessageRepository {
	return &MessageRepository{pool: pool}
}

func (r *MessageRepository) Create(ctx context.Context, msg *domain.Message) error {
	query := `
		INSERT INTO messages (chat_id, sender_id, content)
		VALUES ($1, $2, $3)
		RETURNING id, created_at	
	`

	err := r.pool.QueryRow(ctx, query, msg.ChatID, msg.SenderID, msg.Content).Scan(&msg.ID, &msg.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (r *MessageRepository) GetByChatID(ctx context.Context, chatID uuid.UUID, limit, offset int) ([]*domain.Message, error) {
	query := `
		SELECT 
			m.id, 
			m.chat_id, 
			m.sender_id, 
			u.username, 
			m.content, 
			m.created_at,
			EXISTS (
				SELECT 1 
				FROM chat_members cm 
				WHERE cm.chat_id = m.chat_id 
				  AND cm.user_id != m.sender_id 
				  AND cm.last_read_at >= m.created_at
			) AS is_read
		FROM messages m
		LEFT JOIN users u ON m.sender_id = u.id
		WHERE m.chat_id = $1
		ORDER BY m.created_at ASC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.pool.Query(ctx, query, chatID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get by chatid: %w", err)
	}
	defer rows.Close()

	var msgs []*domain.Message

	for rows.Next() {
		var msg domain.Message
		err := rows.Scan(
			&msg.ID,
			&msg.ChatID,
			&msg.SenderID,
			&msg.SenderUsername,
			&msg.Content,
			&msg.CreatedAt,
			&msg.IsRead,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan msg row: %w", err)
		}
		msgs = append(msgs, &msg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during rows iteraiton: %w", err)
	}

	return msgs, nil
}
