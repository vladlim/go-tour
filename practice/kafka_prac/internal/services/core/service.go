package core

import (
	"context"
	"kafkaprac/internal/services/example"
	"log/slog"
)

type CoreService interface {
	SendTestEvents(ctx context.Context, count int) error
	StartConsumers(ctx context.Context)
	Close() error
}

type core struct {
	exampleService example.Service
	logger         *slog.Logger
}

type CoreServiceConfig struct {
	ExampleService example.Service
	Logger         *slog.Logger
}

func NewService(cfg CoreServiceConfig) CoreService {
	return &core{
		exampleService: cfg.ExampleService,
		logger:         cfg.Logger,
	}
}

func (c *core) SendTestEvents(ctx context.Context, count int) error {
	return c.exampleService.SendTestEvents(ctx, count)
}

func (c *core) StartConsumers(ctx context.Context) {
	c.exampleService.StartConsumers(ctx)
}

func (c *core) Close() error {
	return c.exampleService.Close()
}