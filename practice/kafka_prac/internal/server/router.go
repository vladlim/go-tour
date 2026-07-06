package server

import (
	"net/http"
)

func (s *Server) setupRouter() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ping", s.pingHandler)
	mux.HandleFunc("POST /events", s.produceEventsHandler)

	s.server.Handler = mux

	return mux
}
