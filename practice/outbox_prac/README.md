# Outbox practice

Учебный проект из двух Go-сервисов для наблюдения за transactional outbox,
retry/backoff, at-least-once delivery, идемпотентностью и eventual consistency.

## Поток данных

```text
POST /events
     │ одна DB-транзакция
     ▼
producer-db: events + outbox
     │ outbox worker, batch + FOR UPDATE SKIP LOCKED
     ▼
Kafka: demo.events
     │ consumer group event-processor
     ▼
consumer-db: processed_events
```

У каждого сервиса собственная БД. Producer не делает сетевой вызов в Kafka из
HTTP-транзакции: он атомарно сохраняет бизнес-событие и намерение его отправить.
Поэтому после ответа `201` запись надежно остается в outbox, даже если Kafka
недоступна.

## Запуск

Нужны Docker с Compose и свободные порты 5432, 5433, 8080, 8081 и 9092.

```bash
make up
make logs                 # в другом терминале
make event                # создать событие
```

Kafka UI: http://localhost:8080. Ручной запрос:

```bash
curl -X POST http://localhost:8081/events \
  -H 'Content-Type: application/json' \
  -d '{"message":"hello outbox"}'
```

Посмотреть состояние обеих баз:

```bash
docker compose exec producer-db psql -U producer -d producer \
  -c 'select id,event_id,attempts,next_attempt_at,published_at,last_error from outbox;'

docker compose exec consumer-db psql -U consumer -d consumer \
  -c 'select * from processed_events;'
```

В `processed_message` исходная строка будет переведена в верхний регистр и
получит суффикс `[PROCESSED]`.

## Эксперименты

### Outbox, retry и exponential backoff

Остановить Kafka, создать событие и наблюдать outbox:

```bash
docker compose stop kafka
make event
docker compose logs -f producer
```

В логах будут попытки с задержкой 1, 2, 4, 8, 16, максимум 30 секунд. В таблице
меняются `attempts`, `next_attempt_at`, `last_error`. Затем:

```bash
docker compose start kafka
```

Запись в итоге получит `published_at`, а сообщение дойдет до consumer.

### Eventual consistency

Остановить consumer, отправить несколько событий, проверить пустую/отстающую
`processed_events`, затем снова запустить consumer:

```bash
docker compose stop consumer
make event
docker compose start consumer
```

Producer и consumer сходятся не мгновенно, а после доставки и обработки.

### At-least-once и идемпотентность

Kafka offset коммитится только после записи в consumer-db. Если процесс упадет
после INSERT, но до commit offset, Kafka выдаст сообщение повторно. Кроме того,
классический polling publisher не может атомарно объединить Kafka publish и
`published_at`: падение между ними тоже дает дубль. Поэтому
`processed_events.source_event_id` — PRIMARY KEY, а INSERT использует
`ON CONFLICT DO NOTHING`.

Это дает **at-least-once delivery**, но фактически одну строку результата на
событие. Exactly-once между произвольной БД и Kafka этот пример не обещает.

## Важные места в коде

- `cmd/producer/main.go:createEvent` — транзакция `events + outbox`.
- `cmd/producer/main.go:publishBatch` — batch polling, блокировка строк,
  публикация, retry/backoff.
- `cmd/consumer/main.go:processWithRetry` — преобразование, идемпотентный INSERT
  и retry записи.
- `migrations/` — схемы независимых баз.

Для полного удаления данных и повторного применения миграций: `make reset`.
