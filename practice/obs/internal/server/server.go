package server

import (
	"fmt"
	"net/http"
	"obs/internal/service"
	"time"
)

type ServerCfg struct {
	Host    string
	Port    int
	Service service.Service
}

type server struct {
	service service.Service
	server  http.Server
}

func New(cfg ServerCfg) *server {
	

	s := &server{
		service: cfg.Service,
		server: http.Server{
			Addr:        fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
			ReadTimeout: 15 * time.Second,
		},
	}

	s.server.
}

func (s *server) setupRouter() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET ")
}