package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/segmentio/kafka-go"

	"outboxprac/internal/model"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := connectDB(ctx, env("DATABASE_URL", "postgres://consumer:consumer@localhost:5433/consumer?sslmode=disable"))
	if err != nil {
		slog.Error("database unavailable", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	r := kafka.NewReader(kafka.ReaderConfig{Brokers: strings.Split(env("KAFKA_BROKERS", "localhost:9092"), ","), Topic: env("KAFKA_TOPIC", "demo.events"), GroupID: env("KAFKA_GROUP_ID", "event-processor"), MinBytes: 1, MaxBytes: 10e6, CommitInterval: 0})
	defer r.Close()

	slog.Info("consumer started")

	for {
		msg, err := r.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			slog.Warn("fetch failed", "error", err)
			continue
		}

		if err := processWithRetry(ctx, db, msg, 5); err != nil {
			slog.Error("message processing exhausted retries; offset not committed", "error", err, "offset", msg.Offset)
			time.Sleep(5 * time.Second)
			continue
		}

		if err := r.CommitMessages(ctx, msg); err != nil {
			slog.Error("offset commit failed; message can be delivered again", "error", err)
			continue
		}
	}
}

func processWithRetry(ctx context.Context, db *pgxpool.Pool, msg kafka.Message, maxAttempts int) error {
	var event model.Event

	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return fmt.Errorf("invalid event: %w", err)
	}

	transformed := strings.ToUpper(event.Message) + " [PROCESSED]"
	
	var err error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		_, err = db.Exec(ctx, `INSERT INTO processed_events (source_event_id, source_event_type, original_message, processed_message, source_occurred_at) VALUES ($1,$2,$3,$4,$5) ON CONFLICT (source_event_id) DO NOTHING`, event.ID, event.Type, event.Message, transformed, event.OccurredAt)
		if err == nil {
			slog.Info("event processed", "event_id", event.ID, "partition", msg.Partition, "offset", msg.Offset)
			return nil
		}
		
		d := backoff(attempt)
		
		slog.Warn("processing failed; retrying", "event_id", event.ID, "attempt", attempt, "backoff", d, "error", err)
		
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(d):
		}
	}

	return err
}

func backoff(attempt int) time.Duration {
	seconds := math.Pow(2, float64(attempt-1))
	if seconds > 30 {
		seconds = 30
	}
	return time.Duration(seconds * float64(time.Second))
}

func connectDB(ctx context.Context, url string) (*pgxpool.Pool, error) {
	var db *pgxpool.Pool
	var err error

	for attempt := 1; attempt <= 10; attempt++ {
		db, err = pgxpool.New(ctx, url)
		if err == nil {
			err = db.Ping(ctx)
		}
		if err == nil {
			return db, nil
		}
		d := backoff(attempt)
		slog.Warn("database connection retry", "attempt", attempt, "backoff", d, "error", err)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(d):
		}
	}
	return nil, err
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
