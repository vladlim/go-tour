package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"observability-demo/internal/api"
	"observability-demo/internal/telemetry"
)

const serviceVersion = "1.0.0"

func main() {
	serviceName := envOrDefault("SERVICE_NAME", "observability-demo")
	environment := envOrDefault("APP_ENV", "local")
	httpAddr := envOrDefault("HTTP_ADDR", ":8080")

	logger := newLogger(serviceName, environment)
	slog.SetDefault(logger)

	startupCtx, startupCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer startupCancel()

	shutdownTracing, err := telemetry.InitTracing(
		startupCtx,
		logger,
		serviceName,
		serviceVersion,
		environment,
	)
	if err != nil {
		logger.Error("failed to initialize tracing", slog.Any("error", err))
		os.Exit(1)
	}

	registry := api.NewRegistry()
	metrics := api.NewMetrics(registry)
	handler := api.NewServer(logger, metrics).Handler(registry)

	httpServer := &http.Server{
		Addr:              httpAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info(
			"HTTP server started",
			slog.String("address", httpAddr),
			slog.String("metrics_path", "/metrics"),
		)

		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-signals:
		logger.Info("shutdown signal received", slog.String("signal", sig.String()))
	case err := <-serverErrors:
		logger.Error("HTTP server failed", slog.Any("error", err))
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown failed", slog.Any("error", err))
	}

	if err := shutdownTracing(shutdownCtx); err != nil {
		logger.Error("tracing shutdown failed", slog.Any("error", err))
	}

	logger.Info("service stopped")
}

func newLogger(serviceName string, environment string) *slog.Logger {
	level := new(slog.LevelVar)
	level.Set(parseLogLevel(envOrDefault("LOG_LEVEL", "debug")))

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     level,
		AddSource: false,
	})

	return slog.New(handler).With(
		slog.String("service", serviceName),
		slog.String("service_version", serviceVersion),
		slog.String("environment", environment),
	)
}

func parseLogLevel(value string) slog.Level {
	switch strings.ToLower(value) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func envOrDefault(key string, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
