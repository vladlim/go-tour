package server

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"
)

func (h *handler) Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()

		h.logger.Info("request started",
			"method", r.Method,
			"path", r.URL.Path,
		)

		next.ServeHTTP(w, r)

		h.logger.Info("request finished",
			"duration", time.Since(started),
		)
	})
}

func (h *handler) Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				if recovered == http.ErrAbortHandler {
					panic(recovered)
				}
				h.logger.ErrorContext(
					r.Context(),
					"HTTP handler panicked",
					slog.Any("panic", recovered),
					slog.String("stack", string(debug.Stack())),
					slog.String("path", r.URL.Path),
				)

				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
