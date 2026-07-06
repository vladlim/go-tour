package kafka

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"
)

type Producer interface {
	SendMessages(ctx context.Context, messages ...Message) error
	Close() error
}

type Message struct {
	Key     []byte
	Value   []byte
	Headers map[string]string
}

type ProducerConfig struct {
	Brokers []string
	Topic   string
	Logger  *slog.Logger
}

type producer struct {
	writer *kafka.Writer
	topic  string
	logger *slog.Logger
}

func NewProducer(cfg ProducerConfig) Producer {
	return &producer{
		topic:  cfg.Topic,
		logger: cfg.Logger,
		writer: &kafka.Writer{
			Addr:         kafka.TCP(cfg.Brokers...),
			Topic:        cfg.Topic,
			Balancer:     &kafka.Hash{}, // важный момент: одинаковый key -> одна partition
			RequiredAcks: kafka.RequireAll,
			Async:        false,
			BatchTimeout: 10 * time.Millisecond,
		},
	}
}

func (p *producer) SendMessages(ctx context.Context, messages ...Message) error {
	kafkaMessages := make([]kafka.Message, 0, len(messages))

	for _, msg := range messages {
		headers := make([]kafka.Header, 0, len(msg.Headers))

		for key, value := range msg.Headers {
			headers = append(headers, kafka.Header{
				Key:   key,
				Value: []byte(value),
			})
		}

		kafkaMessages = append(kafkaMessages, kafka.Message{
			Key:     msg.Key,
			Value:   msg.Value,
			Headers: headers,
			Time:    time.Now(),
		})
	}

	if err := p.writer.WriteMessages(ctx, kafkaMessages...); err != nil {
		return fmt.Errorf("failed to write kafka messages: %w", err)
	}

	p.logger.Info(
		"kafka messages produced",
		slog.String("topic", p.topic),
		slog.Int("count", len(kafkaMessages)),
	)

	return nil
}

func (p *producer) Close() error {
	if err := p.writer.Close(); err != nil {
		return fmt.Errorf("failed to close kafka writer: %w", err)
	}

	return nil
}
