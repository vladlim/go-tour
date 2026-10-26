package telemetry

import (
	"context"
	"errors"
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// InitTracing creates an OTLP/HTTP exporter and installs a global tracer provider.
// The exporter reads standard OpenTelemetry environment variables, including:
//
//	OTEL_EXPORTER_OTLP_ENDPOINT=http://jaeger:4318
//
// The returned function must be called during graceful shutdown so buffered
// spans are flushed before the process exits.
func InitTracing(
	ctx context.Context,
	logger *slog.Logger,
	serviceName string,
	serviceVersion string,
	environment string,
) (func(context.Context) error, error) {
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		logger.Error("OpenTelemetry error", slog.Any("error", err))
	}))

	exporter, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, err
	}

	res, err := resource.New(
		ctx,
		resource.WithAttributes(
			attribute.String("service.name", serviceName),
			attribute.String("service.version", serviceVersion),
			attribute.String("deployment.environment.name", environment),
		),
	)
	if err != nil {
		shutdownErr := exporter.Shutdown(ctx)
		return nil, errors.Join(err, shutdownErr)
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		// AlwaysSample is useful for a local demo. In production this usually
		// needs to be replaced with a lower sampling rate or tail sampling.
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	return provider.Shutdown, nil
}
