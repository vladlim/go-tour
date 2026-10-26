package httpserver

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"
)

func Run(ctx context.Context, addr string, handler http.Handler, shutdownTimeout time.Duration) error {
	srv := &http.Server{Addr: addr, Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second}
	errC := make(chan error, 1)
	go func() { slog.Info("http server started", "addr", addr); errC <- srv.ListenAndServe() }()
	select {
	case err := <-errC:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return err
		}
		return nil
	}
}
