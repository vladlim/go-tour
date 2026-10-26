# PostgreSQL: репликация и шардирование на Go

Небольшая учебная лаборатория, в которой оба понятия можно увидеть руками:

```text
                         Go-приложение
                    user_id % 2 (router)
                       /             \
                      /               \
             shard 0 primary     shard 1 primary   <- запись
                    |                  |
             streaming replica  streaming replica <- чтение
```

- **Шардирование** делит строки: чётные `user_id` находятся на shard 0, нечётные — на shard 1.
- **Физическая streaming replication** PostgreSQL копирует WAL с primary на replica внутри каждого шарда.
- Запись всегда идёт в primary. Обычное чтение идёт в replica.
- Поиск с `user_id` обращается к одному шарду, а `list-users` делает scatter-gather по всем шардам.

Это демонстрационный стенд, не production-шаблон: здесь фиксированы два шарда, пароли открыты, нет автоматического failover, TLS, connection proxy и повторов при временных ошибках.

## Запуск

Нужны Docker с Compose и Go 1.24+.

```bash
cd practice/db_replication
docker compose up -d --wait
go mod download
go run ./cmd status
```

Ключ `--wait` важен: Compose дождётся не только запуска процессов, но и готовности обеих реплик. Разрешение replication-соединений находится в `docker/primary-pg_hba.conf`; для специального подключения `pg_basebackup` требуется отдельная строка с базой `replication`.

Ожидаемый смысл статуса (конкретный lag будет другим):

```text
shard=0 primary_in_recovery=false replica_in_recovery=true replay_lag=...
shard=1 primary_in_recovery=false replica_in_recovery=true replay_lag=...
```

`false` у primary означает, что сервер принимает запись. `true` у replica означает режим восстановления: она применяет WAL и доступна только для чтения.

## Лабораторная работа

### 1. Увидеть шардирование

```bash
go run ./cmd create-user -id 1 -name Alice -email alice@example.com
go run ./cmd create-user -id 2 -name Bob -email bob@example.com
go run ./cmd create-user -id 3 -name Carol -email carol@example.com
```

Маршрутизатор вычисляет `user_id % 2`, поэтому Alice и Carol попадут на shard 1, Bob — на shard 0. Проверьте primary напрямую:

```bash
docker compose exec shard0-primary psql -U app -d app -c 'TABLE users;'
docker compose exec shard1-primary psql -U app -d app -c 'TABLE users;'
```

### 2. Увидеть репликацию

Команда читает пользователя с replica выбранного шарда:

```bash
go run ./cmd get-user 1
```

Сравните с явным чтением с primary:

```bash
go run ./cmd get-user --primary 1
```

Попытка записи на replica должна завершиться ошибкой `cannot execute INSERT in a read-only transaction`:

```bash
docker compose exec shard1-replica psql -U app -d app \
  -c "INSERT INTO users(id,name,email) VALUES(99,'Wrong node','wrong@example.com');"
```

Состояние WAL можно посмотреть на обеих сторонах:

```bash
docker compose exec shard0-primary psql -U app -d app -c \
  "SELECT application_name, state, sent_lsn, replay_lsn FROM pg_stat_replication;"
docker compose exec shard0-replica psql -U app -d app -c \
  "SELECT pg_last_wal_receive_lsn(), pg_last_wal_replay_lsn();"
```

Репликация асинхронная. Сразу после успешного `INSERT` replica теоретически может ещё не видеть строку — это называется **replication lag** и приводит к проблеме read-after-write. Флаг `--primary` показывает простой способ получить строго актуальное чтение после собственной записи.

### 3. Увидеть scatter-gather

```bash
go run ./cmd list-users
```

Приложение запрашивает обе реплики и объединяет ответы. Без shard key невозможно заранее выбрать один сервер. В реальном проекте такие запросы требуют обработки частичных отказов, глобальной сортировки, лимитов и дедлайнов.

### 4. Имитировать отставание реплики

Приостановите применение WAL на shard 0, добавьте чётного пользователя и сравните чтение:

```bash
docker compose exec shard0-replica psql -U app -d app -c 'SELECT pg_wal_replay_pause();'
go run ./cmd create-user -id 4 -name Dave -email dave@example.com
go run ./cmd get-user --primary 4
go run ./cmd get-user 4
```

Последняя команда не найдёт пользователя на replica. Возобновите применение WAL:

```bash
docker compose exec shard0-replica psql -U app -d app -c 'SELECT pg_wal_replay_resume();'
go run ./cmd get-user 4
```

### 5. Имитировать отказ primary

```bash
docker compose stop shard0-primary
docker compose exec shard0-replica psql -U app -d app -c 'SELECT * FROM users WHERE id=2;'
go run ./cmd create-user -id 6 -name Eve -email eve@example.com
```

Сама replica продолжает отдавать уже скопированные данные, но запись не выполнится. Учебный CLI при старте проверяет все четыре соединения и тоже завершится ошибкой — это наглядный пример того, почему production-клиент должен уметь переживать частичные отказы. Автоматического promotion здесь намеренно нет: failover — отдельная задача оркестратора (например, Patroni), а не шард-роутера. Вернуть узел:

```bash
docker compose start shard0-primary
```

## Где что реализовано

- `docker-compose.yml` — два независимых primary и две replica.
- `docker/replica-entrypoint.sh` — первоначальный `pg_basebackup` и создание `standby.signal` через ключ `-R`.
- `internal/shards/manager.go` — формула шардирования и отдельные пулы primary/replica.
- `internal/store/store.go` — write/read routing и scatter-gather.
- `cmd/main.go` — учебный CLI.

DSN можно заменить переменными из `.env.example`. Compose при этом поднимает только базы; переменные нужны Go-процессу.

## Остановка и полный сброс

```bash
docker compose down          # данные остаются в volumes
docker compose down -v       # удалить все учебные данные
```

После изменения init SQL нужен именно `down -v`, потому что PostgreSQL выполняет `/docker-entrypoint-initdb.d` только при создании пустого data directory.

## Важные выводы

1. Шардирование и репликация ортогональны: первое распределяет разные данные, второе копирует одни и те же.
2. Shard key нужен приложению до SQL-запроса; PostgreSQL одного шарда ничего не знает о другом.
3. Связанные данные (`users` и `orders`) должны иметь один shard key, чтобы JOIN и транзакции оставались локальными.
4. Асинхронная replica масштабирует чтение, но допускает устаревшие ответы.
5. Формула `% 2` годится для обучения. Если поменять число шардов, большая часть ключей начнёт указывать не туда; production-системам нужны бакеты, directory mapping или управляемый resharding.
