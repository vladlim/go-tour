package kafka

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

type ConsumedMessage struct {
	Topic     string
	Partition int
	Offset    int64
	Key       []byte
	Value     []byte
	Headers   map[string]string
	Time      time.Time
}

type MessageHandler func(ctx context.Context, msg ConsumedMessage) error

type ConsumerGroup struct {
	readers []*kafka.Reader
	logger  *slog.Logger
}

type ConsumerGroupConfig struct {
	Brokers       []string
	Topic         string
	GroupID       string
	ConsumerCount int
	Logger        *slog.Logger
}

func NewConsumerGroup(cfg ConsumerGroupConfig) *ConsumerGroup {
	consumerCount := cfg.ConsumerCount
	if consumerCount <= 0 {
		consumerCount = 1
	}

	readers := make([]*kafka.Reader, 0, consumerCount)

	for i := 0; i < consumerCount; i++ {
		reader := kafka.NewReader(kafka.ReaderConfig{
			Brokers:        cfg.Brokers,
			Topic:          cfg.Topic,
			GroupID:        cfg.GroupID,
			MinBytes:       1,
			MaxBytes:       10e6,
			MaxWait:        500 * time.Millisecond,
			CommitInterval: 0, // offset коммитим вручную после успешной обработки
			StartOffset:    kafka.FirstOffset,
		})

		readers = append(readers, reader)
	}

	return &ConsumerGroup{
		readers: readers,
		logger:  cfg.Logger,
	}
}

func (g *ConsumerGroup) Start(ctx context.Context, handler MessageHandler) {
	var wg sync.WaitGroup

	for i, reader := range g.readers {
		wg.Add(1)

		go func(consumerID int, r *kafka.Reader) {
			defer wg.Done()

			g.logger.Info(
				"kafka consumer started",
				slog.Int("consumer_id", consumerID),
			)

			for {
				msg, err := r.FetchMessage(ctx)
				if err != nil {
					if errors.Is(err, context.Canceled) {
						g.logger.Info(
							"kafka consumer stopped",
							slog.Int("consumer_id", consumerID),
						)
						return
					}

					g.logger.Error(
						"failed to fetch kafka message",
						slog.Int("consumer_id", consumerID),
						slog.String("error", err.Error()),
					)

					continue
				}

				consumedMessage := toConsumedMessage(msg)

				if err := handler(ctx, consumedMessage); err != nil {
					g.logger.Error(
						"failed to handle kafka message",
						slog.Int("consumer_id", consumerID),
						slog.String("topic", msg.Topic),
						slog.Int("partition", msg.Partition),
						slog.Int64("offset", msg.Offset),
						slog.String("error", err.Error()),
					)

					// ВАЖНО:
					// Offset не коммитим.
					// Значит сообщение может быть прочитано повторно.
					continue
				}

				if err := r.CommitMessages(ctx, msg); err != nil {
					g.logger.Error(
						"failed to commit kafka message",
						slog.Int("consumer_id", consumerID),
						slog.String("topic", msg.Topic),
						slog.Int("partition", msg.Partition),
						slog.Int64("offset", msg.Offset),
						slog.String("error", err.Error()),
					)

					continue
				}

				g.logger.Debug(
					"kafka message committed",
					slog.Int("consumer_id", consumerID),
					slog.String("topic", msg.Topic),
					slog.Int("partition", msg.Partition),
					slog.Int64("offset", msg.Offset),
				)
			}
		}(i+1, reader)
	}

	go func() {
		wg.Wait()
		g.logger.Info("all kafka consumers stopped")
	}()
}

func (g *ConsumerGroup) Close() error {
	var resultErr error

	for _, reader := range g.readers {
		if err := reader.Close(); err != nil {
			resultErr = errors.Join(resultErr, err)
		}
	}

	if resultErr != nil {
		return fmt.Errorf("failed to close kafka readers: %w", resultErr)
	}

	return nil
}

func toConsumedMessage(msg kafka.Message) ConsumedMessage {
	headers := make(map[string]string, len(msg.Headers))

	for _, header := range msg.Headers {
		headers[header.Key] = string(header.Value)
	}

	return ConsumedMessage{
		Topic:     msg.Topic,
		Partition: msg.Partition,
		Offset:    msg.Offset,
		Key:       msg.Key,
		Value:     msg.Value,
		Headers:   headers,
		Time:      msg.Time,
	}
}