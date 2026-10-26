package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/example/outbox-practice/feed/internal/repository"
	"github.com/example/outbox-practice/internal/event"
)

type Consumer struct {
	reader             *kafka.Reader
	repo               *repository.Repository
	retryMin, retryMax time.Duration
	demoCrash          bool
}

func NewConsumer(brokers []string, topic, group string, repo *repository.Repository, retryMin, retryMax time.Duration, demoCrash bool) *Consumer {
	return &Consumer{reader: kafka.NewReader(kafka.ReaderConfig{Brokers: brokers, Topic: topic, GroupID: group, MinBytes: 1, MaxBytes: 10 << 20, CommitInterval: 0}), repo: repo, retryMin: retryMin, retryMax: retryMax, demoCrash: demoCrash}
}
func (c *Consumer) Close() error { return c.reader.Close() }
func (c *Consumer) Run(ctx context.Context) {
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			slog.Error("fetch kafka message", "error", err)
			continue
		}
		var duplicate bool
		for attempt := 0; ; attempt++ {
			duplicate, err = c.process(ctx, msg)
			if err == nil {
				slog.Info("message transaction committed", "event_id", header(msg, "event_id"), "duplicate", duplicate, "partition", msg.Partition, "offset", msg.Offset)
				break
			}
			delay := c.retryMin * time.Duration(1<<min(attempt, 5))
			if delay > c.retryMax {
				delay = c.retryMax
			}
			slog.Error("message processing failed; offset remains uncommitted", "error", err, "retry_in", delay)
			select {
			case <-ctx.Done():
				return
			case <-time.After(delay):
			}
		}
		if c.demoCrash && !duplicate {
			slog.Error("DEMO: crashing after DB commit and before offset commit")
			os.Exit(43)
		}
		for {
			if err := c.reader.CommitMessages(ctx, msg); err == nil {
				break
			} else {
				slog.Error("commit kafka offset failed; retrying before next fetch", "error", err)
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(c.retryMin):
			}
		}
	}
}
func (c *Consumer) process(ctx context.Context, msg kafka.Message) (bool, error) {
	var e event.Envelope[event.PostCreated]
	if err := json.Unmarshal(msg.Value, &e); err != nil {
		return false, fmt.Errorf("decode event (poison message blocks partition): %w", err)
	}
	if e.ID == "" || e.Type != event.PostCreatedV1 {
		return false, fmt.Errorf("unsupported event type %q", e.Type)
	}
	return c.repo.ApplyPostCreated(ctx, e, repository.Metadata{Topic: msg.Topic, Partition: msg.Partition, Offset: msg.Offset})
}
func header(msg kafka.Message, key string) string {
	for _, h := range msg.Headers {
		if h.Key == key {
			return string(h.Value)
		}
	}
	return ""
}
