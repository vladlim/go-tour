# Transactional outbox + inbox

Учебный production-подобный стенд из двух Go-сервисов:

- `core`: HTTP-команда, HTTP idempotency, бизнес-таблица и outbox в одной PostgreSQL-транзакции, relay в Kafka;
- `feed`: Kafka consumer group, inbox и бизнес-эффект в одной PostgreSQL-транзакции, ручной commit offset;
- две независимые PostgreSQL, Kafka в KRaft-режиме и три partition топика `posts.events`.

## Запуск

Нужны Docker с Compose plugin и свободные порты `5432`, `5433`, `8080`, `8081`, `29092`.

```bash
cd practice/outbox
docker compose up --build -d
docker compose ps
docker compose logs -f core feed
```

Миграции встроены в бинарники и применяются при старте. Схемы двух баз намеренно независимы.

## Обычный сценарий

```bash
curl -i -X POST http://localhost:8080/v1/posts \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: request-001' \
  -d '{"title":"My post","body":"Hello, outbox"}'

curl -s http://localhost:8080/v1/posts
curl -s http://localhost:8081/v1/feed
```

Первый ответ — `201 Created`. Повторите тот же POST с тем же ключом и телом: ответ будет `200 OK` и `Idempotency-Replayed: true`, ID поста останется прежним. С тем же ключом, но другим телом получится `409 Conflict`.

`event_id` создаёт core, Kafka key равен `post_id`, а клиентский `Idempotency-Key` в событие не переиспользуется.

## Падение HTTP после commit

```bash
curl -i -X POST 'http://localhost:8080/v1/posts?fail=after_commit' \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: uncertain-result-001' \
  -d '{"title":"Committed, response lost","body":"retry me"}'

curl -i -X POST http://localhost:8080/v1/posts \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: uncertain-result-001' \
  -d '{"title":"Committed, response lost","body":"retry me"}'
```

Первый запрос специально вернёт `500` после commit. Второй вернёт единственный уже созданный пост.

## Дубликат из outbox relay

В `compose.yaml` временно установите для core:

```yaml
DEMO_CRASH_RELAY_ONCE: "true"
```

Пересоздайте core и отправьте новый пост:

```bash
docker compose up -d --build --force-recreate core
docker compose logs -f core feed
```

После Kafka ack core завершится с кодом `42`, не запишет `published_at` и будет перезапущен Compose. Через истечение 30-секундного lease relay опубликует то же событие с тем же `event_id`. Feed увидит дубликат в `inbox_events` и не создаст второй item.

Служебная запись в `demo_failures` делает это падение однократным даже после рестарта. Чтобы повторить опыт с нуля, удалите volumes (`docker compose down -v`).

## Падение consumer до commit offset

Аналогично временно включите у feed:

```yaml
DEMO_CRASH_CONSUMER_ONCE: "true"
```

Feed завершится с кодом `43` после commit своей БД, но до commit Kafka offset. После рестарта Kafka выдаст событие повторно; `PRIMARY KEY (event_id)` в inbox превратит обработку в no-op. На дубликате сервис уже не падает и подтверждает offset.

## Посмотреть таблицы

```bash
docker compose exec core-db psql -U core -d core -c 'table posts'
docker compose exec core-db psql -U core -d core -c 'select id,event_type,attempts,published_at,last_error from outbox_events order by created_at'
docker compose exec core-db psql -U core -d core -c 'select scope,idempotency_key,request_hash,expires_at from idempotency_keys'

docker compose exec feed-db psql -U feed -d feed -c 'table feed_items'
docker compose exec feed-db psql -U feed -d feed -c 'select * from inbox_events order by processed_at'
```

## Retention и ограничения демо

- опубликованный outbox хранится 7 дней; неопубликованные события cleanup не удаляет;
- HTTP idempotency keys хранятся 24 часа;
- inbox хранится 30 дней — дольше Kafka retention/replay-окна, которое в реальном окружении нужно формально согласовать;
- cleanup удаляет записи батчами;
- `UNIQUE(source_post_id)` дополнительно защищает бизнес-модель feed после очистки inbox;
- некорректное/неподдерживаемое сообщение сейчас блокирует partition с retry. В production обычно добавляют наблюдаемую retry/DLQ-политику, но она намеренно не спрятана в этой демонстрации;
- Kafka с одним broker и replication factor 1 подходит только для локальной практики, а не обеспечивает production durability.

Полностью удалить данные:

```bash
docker compose down -v --remove-orphans
```
