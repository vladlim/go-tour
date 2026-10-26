package transport

import (
	"encoding/json"
	"github.com/example/outbox-practice/feed/internal/repository"
	"net/http"
)

type Handler struct{ repo *repository.Repository }

func NewHandler(repo *repository.Repository) http.Handler {
	h := &Handler{repo: repo}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/feed", h.list)
	mux.HandleFunc("GET /health/live", live)
	mux.HandleFunc("GET /health/ready", h.ready)
	return mux
}
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.List(r.Context())
	if err != nil {
		write(w, 500, map[string]string{"error": "internal error"})
		return
	}
	write(w, 200, items)
}
func live(w http.ResponseWriter, _ *http.Request) { write(w, 200, map[string]string{"status": "ok"}) }
func (h *Handler) ready(w http.ResponseWriter, r *http.Request) {
	if err := h.repo.Ping(r.Context()); err != nil {
		write(w, 503, map[string]string{"error": "database unavailable"})
		return
	}
	live(w, r)
}
func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
