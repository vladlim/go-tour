package server

import (
	"log/slog"
	"net/http"
)

type Service interface{}

type handler struct {
	service Service
	logger  *slog.Logger
}

type HandlerConfig struct {
	Service Service
	Logger  *slog.Logger
}

func NewHandler(cfg HandlerConfig) (*handler, error) {
	return &handler{
		service: cfg.Service,
		logger:  cfg.Logger,
	}, nil
}

func (h *handler) Routes() http.Handler {
	router := http.NewServeMux()

	router.HandleFunc("GET /ping", h.Ping)

	return h.Logging(h.Recover(router))
}

func (h *handler) Ping(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("pong"))
}
