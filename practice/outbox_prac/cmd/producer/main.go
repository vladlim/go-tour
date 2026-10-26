package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/segmentio/kafka-go"

	"outboxprac/internal/model"
)

type app struct {
	db           *pgxpool.Pool
	writer       *kafka.Writer
	topic        string
	batch        int
	writeTimeout time.Duration
}

type outboxMessage struct {
	ID       string
	EventID  string
	Payload  []byte
	Attempts int
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := connectDB(ctx, env("DATABASE_URL", "postgres://producer:producer@localhost:5432/producer?sslmode=disable"))
	if err != nil {
		slog.Error("database unavailable", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	brokers := env("KAFKA_BROKERS", "localhost:9092")
	a := &app{
		db:           db,
		topic:        env("KAFKA_TOPIC", "demo.events"),
		batch:        envInt("OUTBOX_BATCH_SIZE", 20),
		writeTimeout: envDuration("KAFKA_WRITE_TIMEOUT", 3*time.Second),
		writer:       &kafka.Writer{Addr: kafka.TCP(brokers), RequiredAcks: kafka.RequireAll, Async: false},
	}
	defer a.writer.Close()

	go a.runOutboxWorker(ctx, envDuration("OUTBOX_POLL_INTERVAL", time.Second))

	mux := http.NewServeMux()
	mux.HandleFunc("POST /events", a.createEvent)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	srv := &http.Server{Addr: env("HTTP_ADDR", ":8081"), Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()
	slog.Info("producer service started", "addr", srv.Addr, "topic", a.topic)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("http server stopped", "error", err)
		os.Exit(1)
	}
}

func (a *app) createEvent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Message == "" {
		http.Error(w, `body must be {"message":"..."}`, http.StatusBadRequest)
		return
	}
	e := model.Event{ID: uuid.NewString(), Type: model.EventType, Message: req.Message, OccurredAt: time.Now().UTC()}
	payload, _ := json.Marshal(e)

	tx, err := a.db.BeginTx(r.Context(), pgx.TxOptions{})
	if err == nil {
		defer tx.Rollback(r.Context())
		_, err = tx.Exec(r.Context(), `INSERT INTO events (id, event_type, message, occurred_at) VALUES ($1,$2,$3,$4)`, e.ID, e.Type, e.Message, e.OccurredAt)
	}
	if err == nil {
		_, err = tx.Exec(r.Context(), `INSERT INTO outbox (id, event_id, topic, payload) VALUES ($1,$2,$3,$4)`, uuid.NewString(), e.ID, a.topic, payload)
	}
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		slog.Error("create event transaction failed", "error", err)
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(e)
}

func (a *app) runOutboxWorker(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if err := a.publishBatch(ctx); err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("outbox batch failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (a *app) publishBatch(ctx context.Context) error {
	tx, err := a.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `SELECT id, event_id, payload, attempts FROM outbox WHERE published_at IS NULL AND next_attempt_at <= now() ORDER BY created_at FOR UPDATE SKIP LOCKED LIMIT $1`, a.batch)
	if err != nil {
		return err
	}
	var batch []outboxMessage
	for rows.Next() {
		var m outboxMessage
		if err := rows.Scan(&m.ID, &m.EventID, &m.Payload, &m.Attempts); err != nil {
			rows.Close()
			return err
		}
		batch = append(batch, m)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	for _, m := range batch {
		writeCtx, cancel := context.WithTimeout(ctx, a.writeTimeout)
		err = a.writer.WriteMessages(writeCtx, kafka.Message{Topic: a.topic, Key: []byte(m.EventID), Value: m.Payload, Headers: []kafka.Header{{Key: "outbox_id", Value: []byte(m.ID)}}})
		cancel()
		if err != nil {
			delay := backoff(m.Attempts + 1)
			_, updateErr := tx.Exec(ctx, `UPDATE outbox SET attempts=attempts+1, last_error=$2, next_attempt_at=now()+$3::interval WHERE id=$1`, m.ID, err.Error(), fmt.Sprintf("%f seconds", delay.Seconds()))
			if updateErr != nil {
				return updateErr
			}
			slog.Warn("publish failed; scheduled retry", "event_id", m.EventID, "attempt", m.Attempts+1, "backoff", delay, "error", err)
			continue
		}
		if _, err = tx.Exec(ctx, `UPDATE outbox SET published_at=now(), attempts=attempts+1, last_error=NULL WHERE id=$1`, m.ID); err != nil {
			return err
		}
		slog.Info("outbox event published", "event_id", m.EventID, "attempt", m.Attempts+1)
	}
	return tx.Commit(ctx)
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
func envInt(key string, fallback int) int {
	v, err := strconv.Atoi(os.Getenv(key))
	if err == nil && v > 0 {
		return v
	}
	return fallback
}
func envDuration(key string, fallback time.Duration) time.Duration {
	v, err := time.ParseDuration(os.Getenv(key))
	if err == nil {
		return v
	}
	return fallback
}
