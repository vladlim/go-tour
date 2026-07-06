package main

import (
	"context"
	"flag"
	"fmt"
	"kafkaprac/internal/config"
	kafkaclient "kafkaprac/internal/kafka"
	"kafkaprac/internal/logger"
	"kafkaprac/internal/server"
	"kafkaprac/internal/services/core"
	"kafkaprac/internal/services/example"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() {
	cfg, err := CfgParse()
	if err != nil {
		panic(err)
	}

	logger, err := logger.New(&logger.LoggerConfig{
		Env: cfg.Env,
	})
	if err != nil {
		panic(err)
	}

	appCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	producer := kafkaclient.NewProducer(kafkaclient.ProducerConfig{
		Brokers: cfg.Kafka.Brokers,
		Topic:   cfg.Kafka.Topic,
		Logger:  logger,
	})

	consumerGroup := kafkaclient.NewConsumerGroup(kafkaclient.ConsumerGroupConfig{
		Brokers:       cfg.Kafka.Brokers,
		Topic:         cfg.Kafka.Topic,
		GroupID:       cfg.Kafka.GroupID,
		ConsumerCount: cfg.Kafka.ConsumerCount,
		Logger:        logger,
	})

	exampleService := example.NewService(example.Config{
		Producer:      producer,
		ConsumerGroup: consumerGroup,
		Logger:        logger,
	})

	coreService := core.NewService(core.CoreServiceConfig{
		ExampleService: exampleService,
		Logger:         logger,
	})

	coreService.StartConsumers(appCtx)

	httpServer := server.NewServer(server.ServerConfig{
		Host:    cfg.Server.Host,
		Port:    cfg.Server.Port,
		Service: coreService,
		Logger:  logger,
	})

	errCh := make(chan error, 1)

	go func() {
		errCh <- httpServer.Start()
	}()

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-stopCh:
		logger.Info("shutdown signal received", slog.String("signal", sig.String()))
	case err := <-errCh:
		if err != nil {
			logger.Error("server stopped with error", slog.String("error", err.Error()))
		}
	}

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("failed to shutdown http server", slog.String("error", err.Error()))
	}

	if err := coreService.Close(); err != nil {
		logger.Error("failed to close core service", slog.String("error", err.Error()))
	}

	logger.Info("application stopped")
}

func CfgParse() (*config.Config, error) {
	cfgPath := flag.String("cfg_path", "", "path to config file")
	flag.Parse()
	flag.Args()

	if strings.TrimSpace(*cfgPath) == "" {
		return nil, fmt.Errorf("config path is required")
	}

	cfg, err := config.ParseConfig(*cfgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return cfg, nil
}
