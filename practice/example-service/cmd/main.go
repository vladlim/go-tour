package main

import (
	"context"
	"errors"
	"example-service/internal/config"
	"example-service/internal/logger"
	"example-service/internal/server"
	"example-service/internal/service"
	"example-service/internal/storage"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if err := run(); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.MustLoad()

	logger, err := logger.New(logger.Config{Env: cfg.Env})
	if err != nil {
		return fmt.Errorf("init logger: %w", err)
	}

	appCtx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	db, err := storage.New(appCtx, storage.Config{
		Username:       cfg.Storage.Username,
		Password:       cfg.Storage.Password,
		Host:           cfg.Storage.Host,
		Database:       cfg.Storage.Database,
		Port:           cfg.Storage.Port,
		SSLMode:        cfg.Storage.SSLMode,
		MaxConns:       cfg.Storage.MaxConns,
		ConnectTimeout: cfg.Storage.ConnectTimeout,
		Logger:         logger,
	})

	if err != nil {
		return fmt.Errorf("init db: %w", err)
	}

	defer db.Close()

	service, err := service.New(service.Config{
		Logger:  logger,
		Storage: db,
	})
	if err != nil {
		return fmt.Errorf("init service: %w", err)
	}

	handler, err := server.NewHandler(server.HandlerConfig{
		Service: service,
		Logger:  logger,
	})
	if err != nil {
		return fmt.Errorf("init handler: %w", err)
	}

	server, err := server.New(server.Config{
		Address: cfg.Server.Address,
		Handler: handler.Routes(),
	})
	if err != nil {
		return fmt.Errorf("init server: %w", err)
	}

	errCh := make(chan error, 1)

	go func() {
		logger.Info("Start listening server")
		errCh <- server.Start()
	}()

	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP: %w", err)
		}
	case <-appCtx.Done():
		logger.Info("context cancelled, shutting down")
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown server: %w", err)
	}

	return nil
}
