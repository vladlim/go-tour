package server

import (
	"encoding/json"
	"kafkaprac/internal/server/models"
	"net/http"
)

func (s *Server) pingHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("pong"))
}

func (s *Server) produceEventsHandler(w http.ResponseWriter, r *http.Request) {
	var req models.PostEventsRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.service.SendTestEvents(r.Context(), req.Count); err != nil {
		s.logger.Error("failed to produce events", "error", err.Error())
		http.Error(w, "failed to produce events", http.StatusInternalServerError)
		return
	}

	response := map[string]any{
		"status": "ok",
		"count":  req.Count,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}
