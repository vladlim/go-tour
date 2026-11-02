package logger

import (
	"fmt"
	"log/slog"
	"os"
)

type Config struct {
	Env string
}

const (
	EnvLocal = "local"
	EnvDev   = "dev"
	EnvProd  = "prod"
)

func New(cfg Config) (*slog.Logger, error) {
	switch cfg.Env {
	case EnvLocal:
		return slog.New(slog.NewTextHandler(
			os.Stdout,
			&slog.HandlerOptions{
				AddSource: true,
				Level:     slog.LevelDebug,
			},
		)), nil

	case EnvDev:
		return slog.New(slog.NewJSONHandler(
			os.Stdout,
			&slog.HandlerOptions{
				AddSource: true,
				Level:     slog.LevelError,
			},
		)), nil

	case EnvProd:
		return slog.New(slog.NewJSONHandler(
			os.Stdout,
			&slog.HandlerOptions{
				AddSource: true,
				Level:     slog.LevelError,
			},
		)), nil

	default:
		return nil, fmt.Errorf("incorrect env: %s", cfg.Env)
	}
}
