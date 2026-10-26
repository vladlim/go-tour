package outbox

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/segmentio/kafka-go"
)

type Message struct {
	ID, AggregateID, Topic string
	Payload                []byte
	Attempts               int
}

type Relay struct {
	db        *pgxpool.Pool
	writer    *kafka.Writer
	interval  time.Duration
	batch     int
	timeout   time.Duration
	demoCrash bool
}

func NewRelay(db *pgxpool.Pool, brokers []string, interval time.Duration, batch int, timeout time.Duration, demoCrash bool) *Relay {
	return &Relay{db: db, interval: interval, batch: batch, timeout: timeout, demoCrash: demoCrash, writer: &kafka.Writer{Addr: kafka.TCP(brokers...), RequiredAcks: kafka.RequireAll, Async: false, BatchSize: 1}}
}

func (r *Relay) Close() error { return r.writer.Close() }

func (r *Relay) Run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		if err := r.publishBatch(ctx); err != nil && ctx.Err() == nil {
			slog.Error("outbox relay iteration failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (r *Relay) publishBatch(ctx context.Context) error {
	rows, err := r.db.Query(ctx, `
	WITH candidates AS (
		SELECT id
		FROM outbox_events
		WHERE published_at IS NULL AND next_attempt_at<=now() AND (locked_until IS NULL OR locked_until<now())
		ORDER BY created_at
		FOR UPDATE SKIP LOCKED
		LIMIT $1
	) UPDATE outbox_events o
	SET locked_until=now()+interval '30 seconds', attempts=attempts+1
	FROM candidates c
	WHERE o.id=c.id
	RETURNING o.id,o.aggregate_id,o.topic,o.payload,o.attempts
	`, r.batch)
	if err != nil {
		return fmt.Errorf("claim batch: %w", err)
	}
	var messages []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.AggregateID, &m.Topic, &m.Payload, &m.Attempts); err != nil {
			rows.Close()
			return err
		}
		messages = append(messages, m)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, m := range messages {
		publishCtx, cancel := context.WithTimeout(ctx, r.timeout)
		err := r.writer.WriteMessages(publishCtx, kafka.Message{Topic: m.Topic, Key: []byte(m.AggregateID), Value: m.Payload, Headers: []kafka.Header{{Key: "event_id", Value: []byte(m.ID)}}})
		cancel()
		if err != nil {
			delay := backoff(m.Attempts)
			_, updateErr := r.db.Exec(ctx, `
			UPDATE outbox_events
			SET locked_until=NULL,next_attempt_at=now()+$2::interval,last_error=$3
			WHERE id=$1
			`, m.ID, fmt.Sprintf("%f seconds", delay.Seconds()), err.Error())
			if updateErr != nil {
				return updateErr
			}
			slog.Warn("event publish failed", "event_id", m.ID, "error", err)
			continue
		}
		slog.Info("event acknowledged by kafka", "event_id", m.ID, "attempt", m.Attempts)
		if r.demoCrash {
			var first bool
			err := r.db.QueryRow(ctx, `INSERT INTO demo_failures(name) VALUES('relay-after-publish') ON CONFLICT(name) DO NOTHING RETURNING true`).Scan(&first)
			if err == nil && first {
				slog.Error("DEMO: crashing after Kafka ack and before outbox update", "event_id", m.ID)
				os.Exit(42)
			}
		}
		if _, err := r.db.Exec(ctx, `UPDATE outbox_events SET published_at=now(),locked_until=NULL,last_error=NULL WHERE id=$1`, m.ID); err != nil {
			return fmt.Errorf("mark published: %w", err)
		}
	}
	return nil
}

func backoff(attempt int) time.Duration {
	shift := attempt - 1
	if shift < 0 {
		shift = 0
	}
	if shift > 5 {
		shift = 5
	}
	return time.Duration(1<<shift) * time.Second
}
