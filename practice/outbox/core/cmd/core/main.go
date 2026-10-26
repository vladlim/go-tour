package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/example/outbox-practice/core/internal/outbox"
	"github.com/example/outbox-practice/core/internal/repository"
	"github.com/example/outbox-practice/core/internal/service"
	"github.com/example/outbox-practice/core/internal/transport"
	"github.com/example/outbox-practice/internal/platform/config"
	"github.com/example/outbox-practice/internal/platform/database"
	"github.com/example/outbox-practice/internal/platform/httpserver"
	"github.com/example/outbox-practice/internal/platform/migrate"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	cfg, err := config.LoadCore()
	if err != nil {
		fatal(err)
	}
	db, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		fatal(err)
	}
	defer db.Close()
	if err = migrate.Run(ctx, db, "core"); err != nil {
		fatal(err)
	}
	repo := repository.New(db)
	svc := service.NewPostService(repo, cfg.Topic, cfg.IdempotencyRetention)
	relay := outbox.NewRelay(db, cfg.Brokers, cfg.OutboxPollInterval, cfg.OutboxBatchSize, cfg.PublishTimeout, cfg.DemoCrashRelayOnce)
	defer relay.Close()
	go relay.Run(ctx)
	go cleanup(ctx, repo, cfg.CleanupInterval, cfg.OutboxRetention)
	if err = httpserver.Run(ctx, cfg.HTTPAddr, transport.NewHandler(svc, repo), cfg.ShutdownTimeout); err != nil {
		fatal(err)
	}
}
func cleanup(ctx context.Context, repo *repository.Repository, interval, retention time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			n, m, err := repo.Cleanup(ctx, time.Now().Add(-retention), time.Now(), 1000)
			if err != nil {
				slog.Error("cleanup failed", "error", err)
			} else if n+m > 0 {
				slog.Info("cleanup completed", "outbox", n, "idempotency", m)
			}
		}
	}
}
func fatal(err error) { slog.Error("fatal", "error", err); os.Exit(1) }
