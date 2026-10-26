package main

import (
	"context"
	"github.com/example/outbox-practice/feed/internal/repository"
	"github.com/example/outbox-practice/feed/internal/service"
	"github.com/example/outbox-practice/feed/internal/transport"
	"github.com/example/outbox-practice/internal/platform/config"
	"github.com/example/outbox-practice/internal/platform/database"
	"github.com/example/outbox-practice/internal/platform/httpserver"
	"github.com/example/outbox-practice/internal/platform/migrate"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	cfg, err := config.LoadFeed()
	if err != nil {
		fatal(err)
	}
	db, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		fatal(err)
	}
	defer db.Close()
	if err = migrate.Run(ctx, db, "feed"); err != nil {
		fatal(err)
	}
	repo := repository.New(db)
	consumer := service.NewConsumer(cfg.Brokers, cfg.Topic, cfg.GroupID, repo, cfg.RetryMin, cfg.RetryMax, cfg.DemoCrashOnce)
	defer consumer.Close()
	go consumer.Run(ctx)
	go cleanup(ctx, repo, cfg.CleanupInterval, cfg.InboxRetention)
	if err = httpserver.Run(ctx, cfg.HTTPAddr, transport.NewHandler(repo), cfg.ShutdownTimeout); err != nil {
		fatal(err)
	}
}
func cleanup(ctx context.Context, repo *repository.Repository, interval, retention time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			n, err := repo.Cleanup(ctx, time.Now().Add(-retention), 1000)
			if err != nil {
				slog.Error("inbox cleanup failed", "error", err)
			} else if n > 0 {
				slog.Info("inbox cleanup completed", "rows", n)
			}
		}
	}
}
func fatal(err error) { slog.Error("fatal", "error", err); os.Exit(1) }
