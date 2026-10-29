package events

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

// MessageCreatedEvent — событие, которое мы шлем в шину Kafka
type MessageCreatedEvent struct {
	ID             uuid.UUID   `json:"id"`
	ChatID         uuid.UUID   `json:"chat_id"`
	SenderID       *uuid.UUID  `json:"sender_id"`
	SenderUsername string      `json:"sender_username"`
	Content        string      `json:"content"`
	RecipientIDs   []uuid.UUID `json:"recipient_ids"`
	CreatedAt      time.Time   `json:"created_at"`
}

// Producer публикует события в Kafka/Redpanda
type Producer struct {
	writer *kafka.Writer
}

func NewProducer(brokers []string, topic string) *Producer {
	return &Producer{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafka.Hash{}, // партиционирование по chat_id
			BatchTimeout: 10 * time.Millisecond,
		},
	}
}

func (p *Producer) PublishMessageCreated(ctx context.Context, event *MessageCreatedEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(event.ChatID.String()), // Ключ партиции: chat_id!
		Value: data,
		Time:  event.CreatedAt,
	})
}

func (p *Producer) Close() error {
	return p.writer.Close()
}
