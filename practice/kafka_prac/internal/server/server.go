package server

import (
	"context"
	"fmt"
	"kafkaprac/internal/services/core"
	"log/slog"
	"net/http"
)

type ServerConfig struct {
	Host    string
	Port    int
	Service core.CoreService
	Logger  *slog.Logger
}

type Server struct {
	server  *http.Server
	service core.CoreService
	logger  *slog.Logger
}

func NewServer(cfg ServerConfig) *Server {
	s := &Server{
		server: &http.Server{
			Addr: fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		},
		service: cfg.Service,
		logger:  cfg.Logger,
	}

	s.server.Handler = s.setupRouter()

	return s
}

func (s *Server) Start() error {
	s.logger.Info("server started", slog.String("address", s.server.Addr))

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("server shutdown started")
	return s.server.Shutdown(ctx)
}
