package example

import (
	"context"
	"encoding/json"
	"fmt"
	kafkaclient "kafkaprac/internal/kafka"
	"log/slog"
	"time"
)

type Service interface {
	SendTestEvents(ctx context.Context, count int) error
	StartConsumers(ctx context.Context)
	Close() error
}

type Config struct {
	Producer      kafkaclient.Producer
	ConsumerGroup *kafkaclient.ConsumerGroup
	Logger        *slog.Logger
}

type service struct {
	producer      kafkaclient.Producer
	consumerGroup *kafkaclient.ConsumerGroup
	logger        *slog.Logger
}

type TestEvent struct {
	EventID   string    `json:"event_id"`
	EventType string    `json:"event_type"`
	EntityID  string    `json:"entity_id"`
	Number    int       `json:"number"`
	CreatedAt time.Time `json:"created_at"`
}

func NewService(cfg Config) Service {
	return &service{
		producer:      cfg.Producer,
		consumerGroup: cfg.ConsumerGroup,
		logger:        cfg.Logger,
	}
}

func (s *service) SendTestEvents(ctx context.Context, count int) error {
	if count <= 0 {
		count = 1
	}

	messages := make([]kafkaclient.Message, 0, count)

	for i := 1; i <= count; i++ {
		// Специально делаем несколько повторяющихся entity_id.
		// Так можно увидеть, что сообщения с одинаковым key
		// будут попадать в одну и ту же partition.
		entityID := fmt.Sprintf("task-%d", i%3)

		event := TestEvent{
			EventID:   fmt.Sprintf("event-%d-%d", time.Now().UnixNano(), i),
			EventType: "task.created",
			EntityID:  entityID,
			Number:    i,
			CreatedAt: time.Now().UTC(),
		}

		payload, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("failed to marshal test event: %w", err)
		}

		messages = append(messages, kafkaclient.Message{
			Key:   []byte(entityID),
			Value: payload,
			Headers: map[string]string{
				"source":        "kafka-prac-api",
				"event_version": "1",
			},
		})
	}

	if err := s.producer.SendMessages(ctx, messages...); err != nil {
		return fmt.Errorf("failed to send test events: %w", err)
	}

	return nil
}

func (s *service) StartConsumers(ctx context.Context) {
	s.consumerGroup.Start(ctx, s.handleMessage)
}

func (s *service) Close() error {
	return s.producer.Close()
}

func (s *service) handleMessage(ctx context.Context, msg kafkaclient.ConsumedMessage) error {
	var event TestEvent

	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return fmt.Errorf("failed to unmarshal kafka event: %w", err)
	}

	s.logger.Info(
		"kafka event consumed",
		slog.String("topic", msg.Topic),
		slog.Int("partition", msg.Partition),
		slog.Int64("offset", msg.Offset),
		slog.String("key", string(msg.Key)),
		slog.String("event_id", event.EventID),
		slog.String("event_type", event.EventType),
		slog.String("entity_id", event.EntityID),
		slog.Int("number", event.Number),
	)

	return nil
}