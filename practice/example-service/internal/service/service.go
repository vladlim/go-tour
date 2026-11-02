package service

import "log/slog"

type service struct {
	logger  *slog.Logger
	storage storage
}

type storage interface{}

type Config struct {
	Logger  *slog.Logger
	Storage storage
}

func New(cfg Config) (*service, error) {
	return &service{
		logger:  cfg.Logger,
		storage: cfg.Storage,
	}, nil
}
