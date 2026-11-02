package server

import (
	"context"
	"net/http"
	"time"
)

type server struct {
	httpServer *http.Server
}

type Config struct {
	Address string
	Handler http.Handler
}

func New(cfg Config) (*server, error) {
	return &server{
		httpServer: &http.Server{
			Addr:              cfg.Address,
			Handler:           cfg.Handler,
			ReadHeaderTimeout: 5 * time.Second,
			IdleTimeout:       60 * time.Second,
		},
	}, nil
}

func (s *server) Start() error {
	return s.httpServer.ListenAndServe()
}

func (s *server) Shutdown(ctx context.Context) error {
	if err := s.httpServer.Shutdown(ctx); err != nil {
		_ = s.httpServer.Close()
		return err
	}
	return nil
}
