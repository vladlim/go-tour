package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Core struct {
	HTTPAddr             string
	DatabaseURL          string
	Brokers              []string
	Topic                string
	OutboxPollInterval   time.Duration
	OutboxBatchSize      int
	PublishTimeout       time.Duration
	ShutdownTimeout      time.Duration
	CleanupInterval      time.Duration
	OutboxRetention      time.Duration
	IdempotencyRetention time.Duration
	DemoCrashRelayOnce   bool
}

type Feed struct {
	HTTPAddr        string
	DatabaseURL     string
	Brokers         []string
	Topic           string
	GroupID         string
	ShutdownTimeout time.Duration
	RetryMin        time.Duration
	RetryMax        time.Duration
	CleanupInterval time.Duration
	InboxRetention  time.Duration
	DemoCrashOnce   bool
}

func LoadCore() (Core, error) {
	c := Core{
		HTTPAddr: env("HTTP_ADDR", ":8080"), DatabaseURL: os.Getenv("DATABASE_URL"),
		Brokers: csv(env("KAFKA_BROKERS", "localhost:9092")), Topic: env("KAFKA_TOPIC", "posts.events"),
		OutboxPollInterval: duration("OUTBOX_POLL_INTERVAL", time.Second), OutboxBatchSize: integer("OUTBOX_BATCH_SIZE", 50),
		PublishTimeout: duration("KAFKA_PUBLISH_TIMEOUT", 5*time.Second), ShutdownTimeout: duration("SHUTDOWN_TIMEOUT", 10*time.Second),
		CleanupInterval: duration("CLEANUP_INTERVAL", time.Hour), OutboxRetention: duration("OUTBOX_RETENTION", 168*time.Hour),
		IdempotencyRetention: duration("IDEMPOTENCY_RETENTION", 24*time.Hour), DemoCrashRelayOnce: boolean("DEMO_CRASH_RELAY_ONCE", false),
	}
	if c.DatabaseURL == "" {
		return c, fmt.Errorf("DATABASE_URL is required")
	}
	return c, nil
}

func LoadFeed() (Feed, error) {
	c := Feed{
		HTTPAddr: env("HTTP_ADDR", ":8080"), DatabaseURL: os.Getenv("DATABASE_URL"),
		Brokers: csv(env("KAFKA_BROKERS", "localhost:9092")), Topic: env("KAFKA_TOPIC", "posts.events"), GroupID: env("KAFKA_GROUP_ID", "feed-v1"),
		ShutdownTimeout: duration("SHUTDOWN_TIMEOUT", 10*time.Second), RetryMin: duration("CONSUMER_RETRY_MIN", time.Second),
		RetryMax: duration("CONSUMER_RETRY_MAX", 30*time.Second), CleanupInterval: duration("CLEANUP_INTERVAL", time.Hour),
		InboxRetention: duration("INBOX_RETENTION", 720*time.Hour), DemoCrashOnce: boolean("DEMO_CRASH_CONSUMER_ONCE", false),
	}
	if c.DatabaseURL == "" {
		return c, fmt.Errorf("DATABASE_URL is required")
	}
	return c, nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
func csv(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
func duration(key string, fallback time.Duration) time.Duration {
	v, err := time.ParseDuration(os.Getenv(key))
	if err == nil && v > 0 {
		return v
	}
	return fallback
}
func integer(key string, fallback int) int {
	v, err := strconv.Atoi(os.Getenv(key))
	if err == nil && v > 0 {
		return v
	}
	return fallback
}
func boolean(key string, fallback bool) bool {
	v, err := strconv.ParseBool(os.Getenv(key))
	if err == nil {
		return v
	}
	return fallback
}
