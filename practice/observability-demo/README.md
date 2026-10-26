# Observability demo for Go

A minimal Go HTTP service demonstrating three observability signals:

- structured JSON logs with `log/slog`;
- Prometheus metrics exposed on `/metrics`;
- OpenTelemetry traces exported over OTLP/HTTP to Jaeger.

## Run

```bash
docker compose up --build
```

Open:

- service: http://localhost:8080/health
- Prometheus: http://localhost:9090
- Jaeger: http://localhost:16686

## Generate requests

Successful request:

```bash
curl "http://localhost:8080/work?delay_ms=300"
```

Slow request:

```bash
curl "http://localhost:8080/work?delay_ms=2500"
```

Failed request:

```bash
curl -i "http://localhost:8080/work?delay_ms=300&fail=true"
```

Several requests:

```bash
for i in $(seq 1 20); do
  curl -s "http://localhost:8080/work?delay_ms=$((i * 50))" > /dev/null
done
```

View logs:

```bash
docker compose logs -f app
```

## PromQL examples

Request rate:

```promql
sum by (route, status) (rate(demo_http_requests_total[1m]))
```

Error rate:

```promql
sum(rate(demo_http_requests_total{status=~"5.."}[1m]))
/
sum(rate(demo_http_requests_total[1m]))
```

P95 latency:

```promql
histogram_quantile(
  0.95,
  sum by (le, route) (rate(demo_http_request_duration_seconds_bucket[5m]))
)
```

Requests currently being processed:

```promql
sum by (route) (demo_http_requests_in_flight)
```

Business operation results:

```promql
sum by (result) (demo_work_operations_total)
```

## Trace-log correlation

The `/work` response contains `trace_id`. The same identifier appears in:

- JSON logs from `docker compose logs app`;
- the corresponding trace in Jaeger.

This lets you move from a failed trace to its logs, or from an error log to the full request timeline.
