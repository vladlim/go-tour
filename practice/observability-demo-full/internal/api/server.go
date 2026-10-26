package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const instrumentationName = "observability-demo/internal/api"

type Metrics struct {
	httpRequestsTotal    *prometheus.CounterVec
	httpRequestDuration  *prometheus.HistogramVec
	httpRequestsInFlight *prometheus.GaugeVec
	workOperationsTotal  *prometheus.CounterVec
}

func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		httpRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "demo",
				Subsystem: "http",
				Name:      "requests_total",
				Help:      "Total number of HTTP requests processed by the service.",
			},
			[]string{"method", "route", "status"},
		),
		httpRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "demo",
				Subsystem: "http",
				Name:      "request_duration_seconds",
				Help:      "HTTP request duration in seconds.",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"method", "route"},
		),
		httpRequestsInFlight: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "demo",
				Subsystem: "http",
				Name:      "requests_in_flight",
				Help:      "Current number of HTTP requests being processed.",
			},
			[]string{"method", "route"},
		),
		workOperationsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "demo",
				Subsystem: "work",
				Name:      "operations_total",
				Help:      "Total number of demo work operations by result.",
			},
			[]string{"result"},
		),
	}

	reg.MustRegister(
		m.httpRequestsTotal,
		m.httpRequestDuration,
		m.httpRequestsInFlight,
		m.workOperationsTotal,
	)

	return m
}

type Server struct {
	logger  *slog.Logger
	metrics *Metrics
	tracer  trace.Tracer
}

func NewServer(logger *slog.Logger, metrics *Metrics) *Server {
	return &Server{
		logger:  logger,
		metrics: metrics,
		tracer:  otel.Tracer(instrumentationName),
	}
}

func NewRegistry() *prometheus.Registry {
	reg := prometheus.NewRegistry()
	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	return reg
}

func (s *Server) Handler(reg prometheus.Gatherer) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("/health", s.instrument("/health", http.HandlerFunc(s.handleHealth)))
	mux.Handle("/work", s.instrument("/work", http.HandlerFunc(s.handleWork)))

	// Prometheus scrapes this endpoint. We deliberately do not wrap /metrics
	// in our own request metrics, otherwise Prometheus's own scrapes add noise.
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	return mux
}

func (s *Server) instrument(route string, next http.Handler) http.Handler {
	observed := s.observabilityMiddleware(route, next)

	// otelhttp must be outside our middleware: it creates the server span first,
	// so the inner logger can read trace_id and span_id from request.Context().
	return otelhttp.NewHandler(
		observed,
		"HTTP "+route,
		otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
			return r.Method + " " + route
		}),
	)
}

func (s *Server) observabilityMiddleware(route string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		recorder := &responseRecorder{ResponseWriter: w}
		logger := loggerWithTrace(r.Context(), s.logger)

		s.metrics.httpRequestsInFlight.WithLabelValues(r.Method, route).Inc()
		logger.InfoContext(
			r.Context(),
			"HTTP request started",
			slog.String("http.method", r.Method),
			slog.String("http.route", route),
		)

		defer func() {
			if recovered := recover(); recovered != nil {
				err := fmt.Errorf("panic: %v", recovered)
				span := trace.SpanFromContext(r.Context())
				span.RecordError(err)
				span.SetStatus(codes.Error, "panic")

				logger.ErrorContext(
					r.Context(),
					"panic recovered",
					slog.Any("panic", recovered),
					slog.String("stack", string(debug.Stack())),
				)

				if !recorder.wroteHeader {
					writeJSON(recorder, http.StatusInternalServerError, map[string]string{
						"error": "internal server error",
					})
				}
			}

			status := recorder.status
			if status == 0 {
				status = http.StatusOK
			}

			duration := time.Since(startedAt)
			s.metrics.httpRequestsInFlight.WithLabelValues(r.Method, route).Dec()
			s.metrics.httpRequestsTotal.WithLabelValues(
				r.Method,
				route,
				strconv.Itoa(status),
			).Inc()
			s.metrics.httpRequestDuration.WithLabelValues(r.Method, route).Observe(duration.Seconds())

			logFn := logger.InfoContext
			if status >= http.StatusInternalServerError {
				logFn = logger.ErrorContext
			} else if status >= http.StatusBadRequest {
				logFn = logger.WarnContext
			}

			logFn(
				r.Context(),
				"HTTP request finished",
				slog.String("http.method", r.Method),
				slog.String("http.route", route),
				slog.Int("http.status_code", status),
				slog.Int("http.response_size_bytes", recorder.bytesWritten),
				slog.Duration("duration", duration),
			)
		}()

		next.ServeHTTP(recorder, r)
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleWork(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	delay, shouldFail, err := parseWorkQuery(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	ctx, span := s.tracer.Start(r.Context(), "work.pipeline")
	defer span.End()

	span.SetAttributes(
		attribute.Int("demo.delay_ms", int(delay.Milliseconds())),
		attribute.Bool("demo.should_fail", shouldFail),
	)

	logger := loggerWithTrace(ctx, s.logger)
	logger.InfoContext(
		ctx,
		"work operation started",
		slog.Duration("requested_delay", delay),
		slog.Bool("should_fail", shouldFail),
	)

	if err := s.validateInput(ctx, delay); err != nil {
		s.finishWorkWithError(ctx, span, err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	if err := s.loadData(ctx); err != nil {
		s.finishWorkWithError(ctx, span, err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load data failed"})
		return
	}

	s.processData(ctx, delay)

	if shouldFail {
		err := errors.New("simulated business operation failure")
		s.finishWorkWithError(ctx, span, err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	s.metrics.workOperationsTotal.WithLabelValues("success").Inc()
	logger.InfoContext(ctx, "work operation completed")

	spanContext := trace.SpanContextFromContext(ctx)
	writeJSON(w, http.StatusOK, map[string]any{
		"status":   "ok",
		"delay_ms": delay.Milliseconds(),
		"trace_id": spanContext.TraceID().String(),
	})
}

func (s *Server) validateInput(ctx context.Context, delay time.Duration) error {
	ctx, span := s.tracer.Start(ctx, "validate.input")
	defer span.End()

	time.Sleep(15 * time.Millisecond)
	if delay < 0 || delay > 5*time.Second {
		err := errors.New("delay_ms must be between 0 and 5000")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	loggerWithTrace(ctx, s.logger).DebugContext(ctx, "input validated")
	return nil
}

func (s *Server) loadData(ctx context.Context) error {
	ctx, span := s.tracer.Start(ctx, "repository.load-data")
	defer span.End()

	span.SetAttributes(attribute.Bool("demo.simulated", true))
	time.Sleep(80 * time.Millisecond)

	loggerWithTrace(ctx, s.logger).DebugContext(
		ctx,
		"simulated repository call finished",
		slog.Duration("duration", 80*time.Millisecond),
	)
	return nil
}

func (s *Server) processData(ctx context.Context, delay time.Duration) {
	ctx, span := s.tracer.Start(ctx, "service.process-data")
	defer span.End()

	span.SetAttributes(attribute.Int64("demo.delay_ms", delay.Milliseconds()))
	time.Sleep(delay)

	loggerWithTrace(ctx, s.logger).DebugContext(
		ctx,
		"simulated processing finished",
		slog.Duration("duration", delay),
	)
}

func (s *Server) finishWorkWithError(ctx context.Context, span trace.Span, err error) {
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	s.metrics.workOperationsTotal.WithLabelValues("error").Inc()
	loggerWithTrace(ctx, s.logger).ErrorContext(
		ctx,
		"work operation failed",
		slog.Any("error", err),
	)
}

func parseWorkQuery(r *http.Request) (time.Duration, bool, error) {
	delayMS := 200
	if raw := r.URL.Query().Get("delay_ms"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil {
			return 0, false, errors.New("delay_ms must be an integer")
		}
		delayMS = value
	}

	shouldFail := false
	if raw := r.URL.Query().Get("fail"); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return 0, false, errors.New("fail must be true or false")
		}
		shouldFail = value
	}

	return time.Duration(delayMS) * time.Millisecond, shouldFail, nil
}

func loggerWithTrace(ctx context.Context, logger *slog.Logger) *slog.Logger {
	spanContext := trace.SpanContextFromContext(ctx)
	if !spanContext.IsValid() {
		return logger
	}

	return logger.With(
		slog.String("trace_id", spanContext.TraceID().String()),
		slog.String("span_id", spanContext.SpanID().String()),
	)
}

type responseRecorder struct {
	http.ResponseWriter
	status       int
	bytesWritten int
	wroteHeader  bool
}

func (r *responseRecorder) WriteHeader(status int) {
	if r.wroteHeader {
		return
	}

	r.status = status
	r.wroteHeader = true
	r.ResponseWriter.WriteHeader(status)
}

func (r *responseRecorder) Write(data []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}

	n, err := r.ResponseWriter.Write(data)
	r.bytesWritten += n
	return n, err
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(value); err != nil {
		// At this point headers may already be written, so the caller cannot
		// reliably replace the response. A production service would log this.
		return
	}
}
