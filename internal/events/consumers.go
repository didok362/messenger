package events

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/didok362/messenger/internal/domain"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

type Broadcaster interface {
	Broadcast(recipientIDs []uuid.UUID, payload any)
}

type MessageSaver interface {
	CreateWithID(ctx context.Context, msg *domain.Message) error
}

// StartWSDeliveryConsumer запускает группу "messenger-ws-delivery"
func StartWSDeliveryConsumer(ctx context.Context, brokers []string, topic string, hub Broadcaster) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        "messenger-ws-delivery",
		MinBytes:       10e3, // 10KB
		MaxBytes:       10e6, // 10MB
		CommitInterval: time.Second,
	})

	log.Println("[Kafka] WS Delivery Consumer started...")

	go func() {
		defer reader.Close()
		for {
			m, err := reader.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Printf("[Kafka] WS Consumer read error: %v\n", err)
				time.Sleep(500 * time.Millisecond)
				continue
			}

			var event MessageCreatedEvent
			if err := json.Unmarshal(m.Value, &event); err != nil {
				log.Printf("[Kafka] WS Consumer unmarshal error: %v\n", err)
				continue
			}

			domainMsg := domain.Message{
				ID:             event.ID,
				ChatID:         event.ChatID,
				SenderID:       event.SenderID,
				SenderUsername: event.SenderUsername,
				Content:        event.Content,
				CreatedAt:      event.CreatedAt,
			}

			hub.Broadcast(event.RecipientIDs, &domainMsg)
		}
	}()
}

// StartDBArchiverConsumer запускает группу "messenger-db-archiver"
func StartDBArchiverConsumer(ctx context.Context, brokers []string, topic string, saver MessageSaver) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		Topic:    topic,
		GroupID:  "messenger-db-archiver",
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})

	log.Println("[Kafka] DB Archiver Consumer started...")

	go func() {
		defer reader.Close()
		for {
			m, err := reader.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Printf("[Kafka] DB Consumer fetch error: %v\n", err)
				time.Sleep(500 * time.Millisecond)
				continue
			}

			var event MessageCreatedEvent
			if err := json.Unmarshal(m.Value, &event); err != nil {
				log.Printf("[Kafka] DB Consumer unmarshal error: %v\n", err)
				_ = reader.CommitMessages(ctx, m)
				continue
			}

			domainMsg := domain.Message{
				ID:             event.ID,
				ChatID:         event.ChatID,
				SenderID:       event.SenderID,
				SenderUsername: event.SenderUsername,
				Content:        event.Content,
				CreatedAt:      event.CreatedAt,
			}

			// Если база упала — пробуем снова и снова, НЕ коммитя смещение в Kafka!
			for {
				err := saver.CreateWithID(ctx, &domainMsg)
				if err == nil {
					break
				}
				log.Printf("[Kafka] Failed to save msg %s to DB (retrying): %v\n", event.ID, err)
				time.Sleep(time.Second)
			}

			// Коммитим смещение (Offset) ТОЛЬКО после успешной записи в базу!
			if err := reader.CommitMessages(ctx, m); err != nil {
				log.Printf("[Kafka] DB Consumer commit error: %v\n", err)
			}
		}
	}()
}
