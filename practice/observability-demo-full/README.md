# Observability Demo: Go + Prometheus + Loki + Alloy + Jaeger + Grafana

## Запуск

Обычный запуск:

```bash
docker compose up --build
```

Запуск с автоматическим генератором трафика:

```bash
docker compose --profile traffic up --build
```

## Интерфейсы

- Go service: http://localhost:8080
- Grafana: http://localhost:3000
- Prometheus: http://localhost:9090
- Jaeger: http://localhost:16686
- Alloy UI: http://localhost:12345
- Loki readiness: http://localhost:3100/ready

Grafana открывается без логина. Готовый dashboard находится в папке **Demo**.

## Ручная генерация данных

```bash
curl "http://localhost:8080/work?delay_ms=300"
curl "http://localhost:8080/work?delay_ms=1500"
curl "http://localhost:8080/work?delay_ms=300&fail=true"
```

PowerShell:

```powershell
Invoke-RestMethod "http://localhost:8080/work?delay_ms=300"
Invoke-RestMethod "http://localhost:8080/work?delay_ms=1500"
try { Invoke-RestMethod "http://localhost:8080/work?delay_ms=300&fail=true" } catch { $_.Exception.Message }
```

## Где смотреть

### Логи

Grafana → **Explore** → datasource **Loki**:

```logql
{service="app"}
```

Или сразу готовый dashboard **Demo / Observability Demo**.

### Метрики

Grafana → готовый dashboard, либо Prometheus:

```promql
demo_http_requests_total
```

```promql
sum by (status) (rate(demo_http_requests_total[1m]))
```

### Трейсы

Jaeger → Service: `observability-demo` → Find Traces.

В Grafana: Explore → datasource **Jaeger** → Search.

## Быстрая диагностика

```bash
docker compose ps
docker compose logs app
docker compose logs prometheus
docker compose logs jaeger
docker compose logs loki
docker compose logs alloy
docker compose logs grafana
```

Проверки:

- http://localhost:8080/metrics должен вернуть текстовые метрики.
- http://localhost:9090/targets должен показывать target `observability-demo` со статусом **UP**.
- http://localhost:3100/ready должен вернуть `ready`.
- Alloy UI на http://localhost:12345 должен показывать компоненты без ошибок.

## Потоки данных

```text
Метрики: Prometheus --GET /metrics--> Go service
Логи:    Go service --stdout--> Docker --Docker API--> Alloy --> Loki
Трейсы:  Go service --OTLP/HTTP--> Jaeger
Просмотр: Grafana --> Prometheus / Loki / Jaeger
```
