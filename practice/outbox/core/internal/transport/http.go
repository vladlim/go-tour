package transport

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/example/outbox-practice/core/internal/repository"
	"github.com/example/outbox-practice/core/internal/service"
)

type Handler struct {
	service *service.PostService
	repo    *repository.Repository
}

func NewHandler(service *service.PostService, repo *repository.Repository) http.Handler {
	h := &Handler{service: service, repo: repo}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/posts", h.create)
	mux.HandleFunc("GET /v1/posts", h.list)
	mux.HandleFunc("GET /health/live", live)
	mux.HandleFunc("GET /health/ready", h.ready)
	return logging(mux)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" || len(key) > 200 {
		problem(w, http.StatusBadRequest, "Idempotency-Key header is required and must be <= 200 chars")
		return
	}
	var req struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil || strings.TrimSpace(req.Title) == "" {
		problem(w, http.StatusBadRequest, "body must contain a non-empty title and optional body")
		return
	}
	p, replayed, err := h.service.Create(r.Context(), key, req.Title, req.Body)
	if errors.Is(err, repository.ErrIdempotencyConflict) {
		problem(w, http.StatusConflict, err.Error())
		return
	}
	if err != nil {
		slog.Error("create post", "error", err)
		problem(w, http.StatusInternalServerError, "internal error")
		return
	}
	if r.URL.Query().Get("fail") == "after_commit" && !replayed {
		problem(w, http.StatusInternalServerError, "demo failure after database commit; retry with the same Idempotency-Key")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if replayed {
		w.Header().Set("Idempotency-Replayed", "true")
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	_ = json.NewEncoder(w).Encode(p)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	posts, err := h.service.List(r.Context())
	if err != nil {
		problem(w, 500, "internal error")
		return
	}
	writeJSON(w, 200, posts)
}
func live(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, 200, map[string]string{"status": "ok"})
}
func (h *Handler) ready(w http.ResponseWriter, r *http.Request) {
	if err := h.repo.Ping(r.Context()); err != nil {
		problem(w, 503, "database unavailable")
		return
	}
	live(w, r)
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func problem(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info("http request", "method", r.Method, "path", r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
