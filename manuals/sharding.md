# Шардирование баз данных

<a id="contents"></a>
## Оглавление

1. [Что такое шардирование](#what-is-sharding)
2. [Шардирование и репликация](#sharding-vs-replication)
3. [Шард, shard key и маршрутизация](#shard-key-routing)
4. [Как выбрать shard key](#choosing-shard-key)
5. [Связанные таблицы и co-location](#related-data)
6. [Таблицы без shard key](#tables-without-key)
7. [Справочники и общие таблицы](#reference-tables)
8. [Внешние ключи и JOIN](#foreign-keys-joins)
9. [Транзакции между шардами](#cross-shard-transactions)
10. [Запросы без shard key: scatter-gather](#scatter-gather)
11. [Глобальные идентификаторы](#global-identifiers)
12. [Стратегии шардирования](#sharding-strategies)
13. [Проблема добавления новых шардов](#resharding)
14. [Виртуальные шарды и бакеты](#virtual-shards)
15. [Плюсы и минусы](#pros-cons)
16. [Когда шардирование действительно нужно](#when-to-shard)
17. [Шардирование и партиционирование](#sharding-vs-partitioning)
18. [Практический пример](#practical-example)
19. [Архитектура примера](#example-architecture)
20. [Схема таблиц](#database-schema)
21. [Docker Compose](#docker-compose)
22. [Конфигурация Go-приложения](#go-config)
23. [Менеджер шардов](#shard-manager)
24. [Создание пользователя и заказа](#create-entities)
25. [Локальная транзакция](#local-transaction)
26. [Чтение связанных данных](#read-related-data)
27. [Поиск без shard key](#lookup-without-key)
28. [Scatter-gather в Go](#scatter-gather-go)
29. [Миграции на всех шардах](#migrations)
30. [Как добавлять новый шард](#add-shard)
31. [Что потребуется в production](#production)
32. [Итог](#summary)

---

<a id="what-is-sharding"></a>
## 1. Что такое шардирование

**Шардирование** — это горизонтальное разделение данных одной логической базы между несколькими физическими базами данных или серверами.

До шардирования:

```text
PostgreSQL
└── users: 1, 2, 3, 4, 5, 6
```

После шардирования:

```text
shard-0:
└── users: 2, 4, 6

shard-1:
└── users: 1, 3, 5
```

Каждый шард хранит **свою часть данных**, а не полную копию всей базы.

Простейшее правило распределения:

```text
shard_id = user_id % 2
```

Тогда:

```text
user_id = 10 → shard-0
user_id = 11 → shard-1
```

Главная идея:

> Шардирование распределяет строки и нагрузку между несколькими независимыми хранилищами.

Обычно шардирование применяют, чтобы масштабировать:

- объем данных;
- нагрузку на запись;
- нагрузку на CPU и диски;
- количество одновременно обслуживаемых клиентов.

[К оглавлению](#contents)

---

<a id="sharding-vs-replication"></a>
## 2. Шардирование и репликация

Эти механизмы решают разные задачи.

### Репликация

При репликации узлы содержат одни и те же данные:

```text
primary:
users 1, 2, 3, 4

replica:
users 1, 2, 3, 4
```

Репликация используется для:

- отказоустойчивости;
- переключения при падении primary;
- распределения нагрузки на чтение;
- резервного копирования.

### Шардирование

При шардировании узлы содержат разные части данных:

```text
shard-0:
users 2, 4

shard-1:
users 1, 3
```

Шардирование используется для:

- распределения общего объема данных;
- распределения записи между несколькими primary;
- горизонтального масштабирования.

### Совместное использование

На практике каждый шард часто имеет собственные реплики:

```text
shard-0
├── primary
└── replica

shard-1
├── primary
└── replica
```

То есть:

- шардирование делит данные;
- репликация копирует данные внутри каждого шарда.

[К оглавлению](#contents)

---

<a id="shard-key-routing"></a>
## 3. Шард, shard key и маршрутизация

### Шард

**Шард** — отдельная физическая база или отдельный узел, который хранит часть общей логической базы.

Шарды обычно имеют одинаковую схему:

```text
shard-0:
users
orders
payments

shard-1:
users
orders
payments
```

Но строки в этих таблицах различаются.

### Shard key

**Shard key** — атрибут, по которому определяется местоположение записи.

Примеры:

```text
user_id
company_id
tenant_id
project_id
region_id
```

### Маршрутизация

Перед выполнением SQL система должна:

1. Получить shard key.
2. Вычислить логический или физический shard ID.
3. Выбрать пул соединений нужной базы.
4. Отправить SQL только на выбранный шард.

```text
request
   ↓
user_id = 15
   ↓
15 % 2 = 1
   ↓
postgres-shard-1
```

При прикладном шардировании обычный PostgreSQL ничего не знает о других шардах. Для каждого инстанса это обычные локальные таблицы.

### Где находится routing-логика

Она может быть реализована:

1. В приложении.
2. В отдельном proxy/router.
3. В coordinator распределенной СУБД.

Для Go + обычного PostgreSQL наиболее понятный учебный вариант — routing на уровне приложения.

[К оглавлению](#contents)

---

<a id="choosing-shard-key"></a>
## 4. Как выбрать shard key

Хороший shard key должен:

1. Присутствовать в большинстве запросов.
2. Равномерно распределять объем данных.
3. Равномерно распределять нагрузку.
4. Позволять хранить связанные данные рядом.
5. Не изменяться в течение жизни сущности.
6. Не создавать один перегруженный шард.

### Пример удачного ключа

Для SaaS-системы часто используют:

```text
tenant_id
company_id
organization_id
```

Все данные одной компании можно хранить вместе:

```text
company
employees
orders
payments
settings
audit_logs
```

### Пример плохого ключа

Распределение по стране:

```text
Russia  → shard-0
Estonia → shard-1
Finland → shard-2
```

Если 80% пользователей относятся к одной стране, `shard-0` получит почти всю нагрузку.

Это называют:

- **data skew** — перекос объема данных;
- **load skew** — перекос нагрузки;
- **hot shard** — горячий шард.

### Важен не только объем

Допустим, пользователи распределены равномерно, но один крупный клиент создает 70% всех заказов. Тогда распределение по числу пользователей выглядит хорошим, но нагрузка остается неравномерной.

Поэтому shard key выбирают по реальным паттернам:

- чтения;
- записи;
- JOIN;
- транзакций;
- роста данных;
- крупных клиентов.

[К оглавлению](#contents)

---

<a id="related-data"></a>
## 5. Связанные таблицы и co-location

Предположим, система шардируется по `user_id`.

Есть таблицы:

```text
users
orders
order_items
payments
sessions
```

Связанные данные желательно помещать на тот же шард, где находится пользователь:

```text
shard(user) = shard(order) = shard(payment)
```

Пример:

```text
user_id = 15
15 % 2 = 1

users(user_id=15)    → shard-1
orders(user_id=15)   → shard-1
payments(user_id=15) → shard-1
sessions(user_id=15) → shard-1
```

Это называется **co-location** — совместное размещение связанных данных.

Преимущества:

- локальные JOIN;
- локальные внешние ключи;
- обычные ACID-транзакции;
- один сетевой запрос к базе;
- меньше межшардовой логики.

Например, запрос выполняется целиком внутри одного PostgreSQL:

```sql
SELECT
    u.id,
    u.name,
    o.id,
    o.amount
FROM users u
JOIN orders o ON o.user_id = u.id
WHERE u.id = $1;
```

Главное правило:

> Таблицы, которые часто соединяются или изменяются в одной транзакции, желательно шардировать по одному ключу.

[К оглавлению](#contents)

---

<a id="tables-without-key"></a>
## 6. Таблицы без shard key

Проблема возникает, если дочерняя таблица не содержит ключ шардирования.

Например:

```sql
CREATE TABLE order_items (
    id bigint PRIMARY KEY,
    order_id bigint NOT NULL,
    product_id bigint NOT NULL
);
```

Если система шардируется по `user_id`, по строке `order_items` нельзя определить нужный шард.

### Вариант 1. Добавить shard key

```sql
CREATE TABLE order_items (
    id bigint PRIMARY KEY,
    order_id bigint NOT NULL,
    user_id bigint NOT NULL,
    product_id bigint NOT NULL
);
```

Теперь:

```text
shard_id = user_id % shard_count
```

`user_id` частично дублируется, но упрощает маршрутизацию.

### Вариант 2. Закодировать shard ID в идентификаторе

Условная структура ID:

```text
[shard_id][timestamp][sequence]
```

Тогда из `order_id` можно получить shard ID.

### Вариант 3. Directory service

Отдельный каталог хранит:

```text
order_id → shard_id
```

Например:

```sql
CREATE TABLE shard_directory (
    entity_type text NOT NULL,
    entity_id uuid NOT NULL,
    shard_id integer NOT NULL,
    PRIMARY KEY (entity_type, entity_id)
);
```

Недостатки:

- дополнительный lookup;
- каталог становится критическим компонентом;
- каталог тоже нужно реплицировать и масштабировать.

### Вариант 4. Найти родительскую сущность

Теоретически:

```text
order_item → order → user_id → shard
```

Но для поиска самого `order` уже нужно знать его шард. Поэтому без каталога или специального ID возникает циклическая проблема.

[К оглавлению](#contents)

---

<a id="reference-tables"></a>
## 7. Справочники и общие таблицы

Не все таблицы обязательно шардировать.

Небольшие и редко изменяемые таблицы можно копировать на каждый шард:

```text
countries
currencies
roles
product_categories
status_dictionary
```

Такие таблицы называют:

- reference tables;
- replicated tables;
- справочниками.

Пример:

```text
shard-0.countries:
1 — Estonia
2 — Finland
3 — Sweden

shard-1.countries:
1 — Estonia
2 — Finland
3 — Sweden
```

Плюсы:

- локальные JOIN;
- простое чтение;
- не нужно обращаться к центральной БД.

Минусы:

- изменения нужно распространять на все шарды;
- возможна временная рассинхронизация.

Другой вариант — хранить общую таблицу отдельно:

```text
user-shard-0
user-shard-1
catalog-db
```

Тогда данные из разных БД объединяет приложение.

[К оглавлению](#contents)

---

<a id="foreign-keys-joins"></a>
## 8. Внешние ключи и JOIN

### Локальные внешние ключи

Если `users` и `orders` находятся на одном шарде, работает обычный внешний ключ:

```sql
CREATE TABLE orders (
    id uuid PRIMARY KEY,
    user_id bigint NOT NULL REFERENCES users(id)
);
```

### Межшардовый внешний ключ

Обычный PostgreSQL не может проверить внешний ключ на другом независимом PostgreSQL-инстансе:

```text
shard-0.orders.user_id
        ↓
shard-1.users.id
```

Межшардовую целостность обеспечивают:

- кодом приложения;
- событиями;
- directory service;
- reconciliation-проверками;
- средствами распределенной СУБД.

### JOIN между шардами

Обычный локальный JOIN между независимыми PostgreSQL не выполняется.

Варианты:

1. Получить данные из первого шарда.
2. Получить связанные данные из второго.
3. Объединить их в Go.

Другие решения:

- coordinator распределенной БД;
- foreign data wrapper;
- денормализация;
- отдельное аналитическое хранилище;
- материализованные представления.

Межшардовые JOIN обычно медленнее и сложнее из-за сетевого обмена и частичных отказов.

[К оглавлению](#contents)

---

<a id="cross-shard-transactions"></a>
## 9. Транзакции между шардами

Локальная транзакция внутри одного шарда работает как обычно:

```sql
BEGIN;

UPDATE users ...;
INSERT INTO orders ...;
INSERT INTO outbox ...;

COMMIT;
```

Если операция затрагивает несколько шардов:

```text
списать деньги на shard-0
зачислить деньги на shard-1
```

одной локальной транзакции недостаточно.

### Варианты решения

#### Распределенная транзакция

Например, двухфазная фиксация.

Недостатки:

- сложность;
- дополнительные блокировки;
- зависимость от coordinator;
- тяжелое восстановление после ошибок.

#### Eventual consistency

Операция разбивается на локальные транзакции:

```text
1. На shard-0 фиксируется списание.
2. Там же в outbox пишется событие.
3. Событие доставляется обработчику shard-1.
4. На shard-1 выполняется зачисление.
```

#### Saga

Для каждого шага задаются:

- локальная транзакция;
- следующее событие;
- идемпотентность;
- компенсация при ошибке.

Главный вывод:

> Хорошая схема шардирования стремится удерживать одну бизнес-операцию внутри одного шарда.

[К оглавлению](#contents)

---

<a id="scatter-gather"></a>
## 10. Запросы без shard key: scatter-gather

Запрос с shard key:

```sql
SELECT *
FROM orders
WHERE user_id = $1;
```

можно направить на один шард.

Запрос без shard key:

```sql
SELECT SUM(amount)
FROM orders
WHERE created_at >= NOW() - INTERVAL '1 day';
```

нужно выполнить на нескольких или всех шардах.

Используется схема **scatter-gather**:

```text
scatter:
отправить запрос на все шарды

gather:
собрать и объединить результаты
```

Пример:

```text
shard-0 → SUM = 500 000
shard-1 → SUM = 700 000

application → 1 200 000
```

Глобальная сортировка сложнее:

```sql
SELECT *
FROM orders
ORDER BY created_at DESC
LIMIT 100;
```

Нужно:

1. Получить верхние строки с каждого шарда.
2. Объединить их.
3. Глобально отсортировать.
4. Взять итоговые 100 строк.

Поэтому глобальную аналитику часто выносят в отдельное хранилище:

```text
PostgreSQL shards
        ↓ events / CDC / ETL
ClickHouse / data warehouse
```

[К оглавлению](#contents)

---

<a id="global-identifiers"></a>
## 11. Глобальные идентификаторы

Если на каждом шарде используется локальный `bigserial`, возможны одинаковые ID:

```text
shard-0: order_id = 100
shard-1: order_id = 100
```

Варианты решения:

### UUID

```sql
id uuid PRIMARY KEY
```

Плюсы:

- генерируется независимо;
- не нужен центральный sequence;
- почти исключены коллизии.

### Snowflake-подобный ID

```text
timestamp + worker_id + sequence
```

Плюсы:

- глобальная уникальность;
- примерно возрастающий порядок;
- можно кодировать worker или shard.

### Разные sequence

Например:

```text
shard-0: 0, 2, 4, 6...
shard-1: 1, 3, 5, 7...
```

### Составной ключ

```text
(shard_id, local_id)
```

Например:

```text
(0, 100)
(1, 100)
```

[К оглавлению](#contents)

---

<a id="sharding-strategies"></a>
## 12. Стратегии шардирования

### Hash sharding

```text
shard_id = hash(shard_key) % shard_count
```

Плюсы:

- обычно равномерное распределение;
- простая маршрутизация.

Минусы:

- изменение количества шардов перемещает много данных;
- диапазонные запросы могут идти на все шарды.

### Range sharding

```text
user_id 1–1 000 000         → shard-0
user_id 1 000 001–2 000 000 → shard-1
```

или по времени:

```text
2025 → shard-0
2026 → shard-1
```

Плюсы:

- удобные диапазонные запросы;
- понятное расположение данных.

Минусы:

- последние диапазоны могут быть горячими;
- возможен перекос.

### Directory-based sharding

Отдельный каталог хранит:

```text
shard_key → shard_id
```

Плюсы:

- гибкое распределение;
- можно переносить отдельного клиента;
- можно выделять крупным клиентам отдельные шарды.

Минусы:

- дополнительный lookup;
- каталог становится критической зависимостью.

### Geographic sharding

```text
EU users   → EU shard
US users   → US shard
Asia users → Asia shard
```

Используется для:

- уменьшения задержки;
- локализации данных;
- региональной изоляции.

### Tenant-based sharding

```text
tenant/company → определенный шард
```

Часто применяется в B2B и SaaS, потому что все данные клиента можно держать вместе.

[К оглавлению](#contents)

---

<a id="resharding"></a>
## 13. Проблема добавления новых шардов

Простая формула:

```text
shard_id = user_id % shard_count
```

плохо переносит изменение количества шардов.

Было:

```text
user_id % 2
```

Стало:

```text
user_id % 3
```

Пример:

```text
user 10:
10 % 2 = 0
10 % 3 = 1
```

После изменения делителя приложение начнет искать пользователя на другом шарде, хотя данные физически не переехали.

Поэтому прямой `% len(shards)` подходит:

- для обучения;
- для прототипа;
- если количество шардов зафиксировано.

Для production используют:

- виртуальные шарды;
- бакеты;
- consistent hashing;
- directory mapping;
- управляемые диапазоны.

[К оглавлению](#contents)

---

<a id="virtual-shards"></a>
## 14. Виртуальные шарды и бакеты

Вместо прямого распределения пользователей по физическим серверам создают много логических бакетов:

```text
bucket_id = hash(user_id) % 1024
```

Затем бакеты назначаются физическим шардам:

```text
buckets 0–511    → shard-0
buckets 512–1023 → shard-1
```

Маршрутизация:

```text
user_id
   ↓ hash
bucket_id
   ↓ routing table
physical shard
```

При добавлении третьего сервера часть бакетов переносится:

```text
buckets 0–340    → shard-0
buckets 341–681  → shard-1
buckets 682–1023 → shard-2
```

Формула вычисления бакета не меняется. Меняется только соответствие:

```text
bucket_id → physical_shard_id
```

Преимущества:

- переносится только часть данных;
- проще балансировать объем и нагрузку;
- горячие бакеты можно перемещать отдельно.

[К оглавлению](#contents)

---

<a id="pros-cons"></a>
## 15. Плюсы и минусы

### Преимущества

- хранение объема данных больше возможностей одного сервера;
- распределение нагрузки на запись;
- распределение CPU, RAM и дисковой нагрузки;
- меньшие локальные индексы;
- изоляция крупных клиентов;
- географическое размещение данных.

### Недостатки

- межшардовые JOIN;
- межшардовые транзакции;
- глобальные агрегаты и сортировки;
- поиск без shard key;
- глобальная уникальность ID;
- сложные миграции;
- сложные backup и restore;
- частичные отказы;
- перебалансировка;
- hot shards;
- более сложные тесты и мониторинг.

Шардирование не ускоряет автоматически любой запрос.

Запрос по shard key обычно становится локальным:

```sql
WHERE user_id = $1
```

Глобальный запрос может стать дороже:

```sql
ORDER BY created_at DESC
LIMIT 100
```

[К оглавлению](#contents)

---

<a id="when-to-shard"></a>
## 16. Когда шардирование действительно нужно

Перед шардированием обычно пробуют:

1. Оптимизировать SQL.
2. Добавить правильные индексы.
3. Использовать `EXPLAIN ANALYZE`.
4. Настроить autovacuum.
5. Устранить bloat.
6. Увеличить CPU, RAM и скорость диска.
7. Использовать connection pooling.
8. Добавить кеширование.
9. Добавить реплики для чтения.
10. Применить партиционирование.
11. Архивировать старые данные.
12. Вынести аналитику в отдельную систему.

Шардирование обосновано, если:

- данные не помещаются на одном узле;
- write-нагрузка превышает возможности одного primary;
- нужна изоляция крупных клиентов;
- требуется географическая локализация;
- рост гарантированно превысит предел вертикального масштабирования.

[К оглавлению](#contents)

---

<a id="sharding-vs-partitioning"></a>
## 17. Шардирование и партиционирование

Оба подхода делят данные на части, но на разных уровнях.

### Партиционирование

Обычно части находятся в одном PostgreSQL-кластере:

```text
PostgreSQL
└── orders
    ├── orders_2025
    ├── orders_2026
    └── orders_2027
```

PostgreSQL сам выбирает партицию.

### Шардирование

Части находятся на разных БД или серверах:

```text
postgres-shard-0
postgres-shard-1
postgres-shard-2
```

Шард выбирает приложение, router или coordinator.

| Характеристика | Партиционирование | Шардирование |
|---|---|---|
| Размещение | Обычно один кластер | Несколько БД/серверов |
| Маршрутизация | PostgreSQL | Приложение/coordinator |
| Масштабирование записи | Один primary | Несколько primary |
| JOIN | Обычные | Между шардами сложнее |
| Транзакции | Локальные | Между шардами сложнее |
| Эксплуатация | Проще | Существенно сложнее |

Партиционирование можно применять внутри каждого шарда:

```text
shard-0.orders_2025
shard-0.orders_2026

shard-1.orders_2025
shard-1.orders_2026
```

[К оглавлению](#contents)

---

<a id="practical-example"></a>
# Практический пример

Рассмотрим прикладное шардирование в Go-приложении с обычным PostgreSQL.

Система хранит:

```text
users
orders
outbox
```

Есть два шарда:

```text
postgres-shard-0
postgres-shard-1
```

Shard key:

```text
user_id
```

Учебное правило:

```text
shard_id = user_id % 2
```

Все заказы пользователя находятся на том же шарде, что и пользователь.

[К оглавлению](#contents)

---

<a id="example-architecture"></a>
## 18. Архитектура примера

```text
                         user_id
                            │
                            ▼
                    Go Shard Manager
                     /             \
                    /               \
                   ▼                 ▼
         PostgreSQL shard-0   PostgreSQL shard-1
         users: 2, 4, 6       users: 1, 3, 5
         orders of them       orders of them
```

Поток создания заказа:

```text
1. Handler получает user_id.
2. Service вызывает ShardManager.ByUserID(user_id).
3. ShardManager вычисляет user_id % 2.
4. Возвращается пул соединений нужного PostgreSQL.
5. Repository выполняет INSERT на выбранном шарде.
```

[К оглавлению](#contents)

---

<a id="database-schema"></a>
## 19. Схема таблиц

Одинаковая миграция применяется на обоих шардах:

```sql
CREATE TABLE users (
    id bigint PRIMARY KEY,
    email text NOT NULL,
    name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE orders (
    id uuid PRIMARY KEY,
    user_id bigint NOT NULL,
    amount bigint NOT NULL CHECK (amount >= 0),
    status text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT orders_user_id_fk
        FOREIGN KEY (user_id)
        REFERENCES users(id)
);

CREATE INDEX orders_user_id_created_at_idx
    ON orders (user_id, created_at DESC);

CREATE TABLE outbox (
    id uuid PRIMARY KEY,
    aggregate_id uuid NOT NULL,
    event_type text NOT NULL,
    payload jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    processed_at timestamptz
);
```

Почему `orders` содержит `user_id`:

- по нему выбирается шард;
- пользователь и заказ co-located;
- доступен локальный JOIN;
- работает локальный внешний ключ;
- доступна обычная транзакция.

`orders.id` — UUID, чтобы ID были уникальны между шардами.

[К оглавлению](#contents)

---

<a id="docker-compose"></a>
## 20. Docker Compose

```yaml
services:
  postgres-shard-0:
    image: postgres:17
    container_name: postgres-shard-0
    environment:
      POSTGRES_USER: app
      POSTGRES_PASSWORD: app
      POSTGRES_DB: app
    ports:
      - "5433:5432"
    volumes:
      - shard_0_data:/var/lib/postgresql/data
      - ./migrations/001_init.sql:/docker-entrypoint-initdb.d/001_init.sql:ro
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U app -d app"]
      interval: 5s
      timeout: 3s
      retries: 10

  postgres-shard-1:
    image: postgres:17
    container_name: postgres-shard-1
    environment:
      POSTGRES_USER: app
      POSTGRES_PASSWORD: app
      POSTGRES_DB: app
    ports:
      - "5434:5432"
    volumes:
      - shard_1_data:/var/lib/postgresql/data
      - ./migrations/001_init.sql:/docker-entrypoint-initdb.d/001_init.sql:ro
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U app -d app"]
      interval: 5s
      timeout: 3s
      retries: 10

volumes:
  shard_0_data:
  shard_1_data:
```

Запуск:

```bash
docker compose up -d
```

Подключения:

```text
shard-0 → localhost:5433
shard-1 → localhost:5434
```

В production миграции обычно запускают отдельным инструментом, а не через `docker-entrypoint-initdb.d`.

[К оглавлению](#contents)

---

<a id="go-config"></a>
## 21. Конфигурация Go-приложения

```yaml
database:
  shards:
    - id: 0
      dsn: postgres://app:app@localhost:5433/app?sslmode=disable

    - id: 1
      dsn: postgres://app:app@localhost:5434/app?sslmode=disable
```

Go-структуры:

```go
type Config struct {
    Database DatabaseConfig `yaml:"database"`
}

type DatabaseConfig struct {
    Shards []ShardConfig `yaml:"shards"`
}

type ShardConfig struct {
    ID  int    `yaml:"id"`
    DSN string `yaml:"dsn"`
}
```

В production DSN лучше получать из secret manager, Vault или Kubernetes Secret.

[К оглавлению](#contents)

---

<a id="shard-manager"></a>
## 22. Менеджер шардов

```go
package shard

import (
    "context"
    "errors"
    "fmt"
    "sort"

    "github.com/jackc/pgx/v5/pgxpool"
)

var (
    ErrNoShards      = errors.New("no database shards configured")
    ErrInvalidUserID = errors.New("invalid user id")
)

type Config struct {
    ID  int
    DSN string
}

type Manager struct {
    pools    map[int]*pgxpool.Pool
    shardIDs []int
}

func NewManager(
    ctx context.Context,
    configs []Config,
) (*Manager, error) {
    if len(configs) == 0 {
        return nil, ErrNoShards
    }

    manager := &Manager{
        pools: make(map[int]*pgxpool.Pool, len(configs)),
    }

    for _, config := range configs {
        if config.ID < 0 {
            manager.Close()
            return nil, fmt.Errorf("invalid shard id: %d", config.ID)
        }

        if _, exists := manager.pools[config.ID]; exists {
            manager.Close()
            return nil, fmt.Errorf("duplicate shard id: %d", config.ID)
        }

        pool, err := pgxpool.New(ctx, config.DSN)
        if err != nil {
            manager.Close()
            return nil, fmt.Errorf(
                "create pool for shard %d: %w",
                config.ID,
                err,
            )
        }

        if err := pool.Ping(ctx); err != nil {
            pool.Close()
            manager.Close()
            return nil, fmt.Errorf(
                "ping shard %d: %w",
                config.ID,
                err,
            )
        }

        manager.pools[config.ID] = pool
        manager.shardIDs = append(manager.shardIDs, config.ID)
    }

    sort.Ints(manager.shardIDs)

    return manager, nil
}

func (m *Manager) Close() {
    for _, pool := range m.pools {
        pool.Close()
    }
}

func (m *Manager) ByUserID(
    userID int64,
) (int, *pgxpool.Pool, error) {
    if userID <= 0 {
        return 0, nil, ErrInvalidUserID
    }

    if len(m.shardIDs) == 0 {
        return 0, nil, ErrNoShards
    }

    index := int(userID % int64(len(m.shardIDs)))
    shardID := m.shardIDs[index]

    pool, ok := m.pools[shardID]
    if !ok {
        return 0, nil, fmt.Errorf(
            "pool for shard %d not found",
            shardID,
        )
    }

    return shardID, pool, nil
}

func (m *Manager) All() map[int]*pgxpool.Pool {
    result := make(map[int]*pgxpool.Pool, len(m.pools))

    for shardID, pool := range m.pools {
        result[shardID] = pool
    }

    return result
}
```

Менеджер:

- создает отдельный пул для каждого шарда;
- проверяет соединения;
- хранит `shard_id → pool`;
- выбирает шард по `user_id`;
- закрывает все пулы при завершении.

Для двух шардов:

```text
user_id = 10 → 10 % 2 = 0 → shard-0
user_id = 11 → 11 % 2 = 1 → shard-1
```

Это учебная схема. В production routing желательно строить через виртуальные бакеты.

[К оглавлению](#contents)

---

<a id="create-entities"></a>
## 23. Создание пользователя и заказа

### Репозиторий пользователей

```go
type UserRepository struct {
    shards *shard.Manager
}

type User struct {
    ID    int64
    Email string
    Name  string
}

func (r *UserRepository) Create(
    ctx context.Context,
    user User,
) error {
    shardID, db, err := r.shards.ByUserID(user.ID)
    if err != nil {
        return fmt.Errorf("select user shard: %w", err)
    }

    _, err = db.Exec(
        ctx,
        `INSERT INTO users (id, email, name)
         VALUES ($1, $2, $3)`,
        user.ID,
        user.Email,
        user.Name,
    )
    if err != nil {
        return fmt.Errorf(
            "insert user into shard %d: %w",
            shardID,
            err,
        )
    }

    return nil
}
```

### Репозиторий заказов

```go
type OrderRepository struct {
    shards *shard.Manager
}

type Order struct {
    ID     uuid.UUID
    UserID int64
    Amount int64
    Status string
}

func (r *OrderRepository) Create(
    ctx context.Context,
    order Order,
) error {
    shardID, db, err := r.shards.ByUserID(order.UserID)
    if err != nil {
        return fmt.Errorf("select order shard: %w", err)
    }

    _, err = db.Exec(
        ctx,
        `INSERT INTO orders (
            id,
            user_id,
            amount,
            status
         )
         VALUES ($1, $2, $3, $4)`,
        order.ID,
        order.UserID,
        order.Amount,
        order.Status,
    )
    if err != nil {
        return fmt.Errorf(
            "insert order into shard %d: %w",
            shardID,
            err,
        )
    }

    return nil
}
```

Заказ маршрутизируется по `UserID`, а не по `Order.ID`.

```text
user 15             → shard-1
orders of user 15   → shard-1
```

[К оглавлению](#contents)

---

<a id="local-transaction"></a>
## 24. Локальная транзакция

При создании заказа требуется:

1. Проверить наличие пользователя.
2. Создать заказ.
3. Создать событие в outbox.

Все операции выполняются на одном шарде:

```go
func (r *OrderRepository) CreateWithOutbox(
    ctx context.Context,
    order Order,
    payload []byte,
) error {
    shardID, db, err := r.shards.ByUserID(order.UserID)
    if err != nil {
        return fmt.Errorf("select shard: %w", err)
    }

    tx, err := db.Begin(ctx)
    if err != nil {
        return fmt.Errorf(
            "begin transaction on shard %d: %w",
            shardID,
            err,
        )
    }
    defer tx.Rollback(ctx)

    var exists bool

    err = tx.QueryRow(
        ctx,
        `SELECT EXISTS(
            SELECT 1
            FROM users
            WHERE id = $1
         )`,
        order.UserID,
    ).Scan(&exists)
    if err != nil {
        return fmt.Errorf("check user: %w", err)
    }

    if !exists {
        return fmt.Errorf(
            "user %d not found on shard %d",
            order.UserID,
            shardID,
        )
    }

    _, err = tx.Exec(
        ctx,
        `INSERT INTO orders (
            id,
            user_id,
            amount,
            status
         )
         VALUES ($1, $2, $3, $4)`,
        order.ID,
        order.UserID,
        order.Amount,
        order.Status,
    )
    if err != nil {
        return fmt.Errorf("insert order: %w", err)
    }

    _, err = tx.Exec(
        ctx,
        `INSERT INTO outbox (
            id,
            aggregate_id,
            event_type,
            payload
         )
         VALUES ($1, $2, $3, $4)`,
        uuid.New(),
        order.ID,
        "order_created",
        payload,
    )
    if err != nil {
        return fmt.Errorf("insert outbox event: %w", err)
    }

    if err := tx.Commit(ctx); err != nil {
        return fmt.Errorf(
            "commit transaction on shard %d: %w",
            shardID,
            err,
        )
    }

    return nil
}
```

Это обычная локальная ACID-транзакция, потому что все таблицы находятся в одном PostgreSQL.

[К оглавлению](#contents)

---

<a id="read-related-data"></a>
## 25. Чтение связанных данных

```go
type UserOrder struct {
    UserID      int64
    UserName    string
    OrderID     uuid.UUID
    OrderAmount int64
    OrderStatus string
}

func (r *UserRepository) GetWithOrders(
    ctx context.Context,
    userID int64,
) ([]UserOrder, error) {
    shardID, db, err := r.shards.ByUserID(userID)
    if err != nil {
        return nil, fmt.Errorf("select shard: %w", err)
    }

    rows, err := db.Query(
        ctx,
        `SELECT
            u.id,
            u.name,
            o.id,
            o.amount,
            o.status
         FROM users u
         JOIN orders o ON o.user_id = u.id
         WHERE u.id = $1
         ORDER BY o.created_at DESC`,
        userID,
    )
    if err != nil {
        return nil, fmt.Errorf(
            "query shard %d: %w",
            shardID,
            err,
        )
    }
    defer rows.Close()

    result := make([]UserOrder, 0)

    for rows.Next() {
        var item UserOrder

        if err := rows.Scan(
            &item.UserID,
            &item.UserName,
            &item.OrderID,
            &item.OrderAmount,
            &item.OrderStatus,
        ); err != nil {
            return nil, fmt.Errorf("scan row: %w", err)
        }

        result = append(result, item)
    }

    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("iterate rows: %w", err)
    }

    return result, nil
}
```

JOIN выполняется локально, потому что пользователь и его заказы находятся на одном шарде.

[К оглавлению](#contents)

---

<a id="lookup-without-key"></a>
## 26. Поиск без shard key

Допустим, нужно найти пользователя по email:

```sql
SELECT *
FROM users
WHERE email = $1;
```

Но шардирование построено по `user_id`.

По email нельзя вычислить:

```text
shard_id = user_id % 2
```

### Решение через directory table

Отдельная центральная таблица:

```sql
CREATE TABLE user_directory (
    email text PRIMARY KEY,
    user_id bigint NOT NULL,
    shard_id integer NOT NULL
);
```

Алгоритм:

```text
1. email → directory DB.
2. Directory возвращает user_id и shard_id.
3. Приложение обращается к нужному шарду.
```

Другие варианты:

- scatter-gather по всем шардам;
- отдельный поисковый индекс;
- требование передавать shard key в API;
- ID, содержащий информацию о шарде.

[К оглавлению](#contents)

---

<a id="scatter-gather-go"></a>
## 27. Scatter-gather в Go

Посчитаем общую выручку по всем шардам:

```go
func TotalRevenue(
    ctx context.Context,
    shards *shard.Manager,
    from time.Time,
) (int64, error) {
    type result struct {
        shardID int
        value   int64
        err     error
    }

    allShards := shards.All()
    resultCh := make(chan result, len(allShards))

    for shardID, db := range allShards {
        shardID := shardID
        db := db

        go func() {
            var value int64

            err := db.QueryRow(
                ctx,
                `SELECT COALESCE(SUM(amount), 0)
                 FROM orders
                 WHERE status = 'paid'
                   AND created_at >= $1`,
                from,
            ).Scan(&value)

            resultCh <- result{
                shardID: shardID,
                value:   value,
                err:     err,
            }
        }()
    }

    var total int64

    for range allShards {
        item := <-resultCh

        if item.err != nil {
            return 0, fmt.Errorf(
                "query shard %d: %w",
                item.shardID,
                item.err,
            )
        }

        total += item.value
    }

    return total, nil
}
```

Проблемы:

- запрос отправляется на каждый шард;
- один медленный шард задерживает ответ;
- один упавший шард может сорвать весь запрос;
- нужно определить поведение при частичном результате;
- стоимость растет вместе с количеством шардов.

Для регулярной аналитики лучше использовать отдельное аналитическое хранилище.

[К оглавлению](#contents)

---

<a id="migrations"></a>
## 28. Миграции на всех шардах

Схема однотипных шардов должна быть совместима.

Например:

```sql
ALTER TABLE orders
ADD COLUMN currency text NOT NULL DEFAULT 'EUR';
```

Миграцию нужно выполнить на каждом шарде.

Пример shell-скрипта:

```bash
#!/usr/bin/env bash

set -euo pipefail

SHARDS=(
  "postgres://app:app@localhost:5433/app?sslmode=disable"
  "postgres://app:app@localhost:5434/app?sslmode=disable"
)

for dsn in "${SHARDS[@]}"; do
  echo "Migrating ${dsn}"
  migrate -path ./migrations -database "${dsn}" up
done
```

В production нужно учитывать:

- частичный провал миграции;
- повторный запуск;
- несовпадение версий схемы;
- rolling deployment;
- блокирующие DDL;
- совместимость старого и нового кода.

Безопасный подход:

```text
1. Расширяющая миграция.
2. Деплой кода, совместимого с обеими схемами.
3. Перенос или заполнение данных.
4. Переключение чтения.
5. Удаление старой структуры отдельной миграцией.
```

[К оглавлению](#contents)

---

<a id="add-shard"></a>
## 29. Как добавлять новый шард

В учебной реализации используется:

```text
user_id % len(shards)
```

Просто добавить третий DSN нельзя.

Неправильный сценарий:

```text
1. Было 2 шарда.
2. Добавили третий.
3. Поменяли % 2 на % 3.
4. Старые данные остались на прежних местах.
5. Приложение ищет их на новых местах.
```

Production-подход с бакетами:

```text
bucket = hash(user_id) % 1024
```

Routing table:

```sql
CREATE TABLE shard_routes (
    bucket_id integer PRIMARY KEY,
    shard_id integer NOT NULL
);
```

Примерный процесс:

```text
1. Поднять новый shard-2.
2. Выбрать часть бакетов для переноса.
3. Скопировать данные.
4. Догнать изменения через CDC или dual write.
5. Переключить routing table.
6. Проверить целостность.
7. Удалить старые копии.
```

Во время переноса нужно отдельно решить:

- куда идут новые записи;
- откуда выполняется чтение;
- как обрабатываются повторы;
- что делать при откате.

[К оглавлению](#contents)

---

<a id="production"></a>
## 30. Что потребуется в production

Учебный пример показывает принцип, но production-решение требует дополнительных механизмов.

### Routing

- виртуальные бакеты;
- versioning routing table;
- кеш маршрутов;
- безопасное обновление конфигурации;
- защита от рассинхронизации.

### Надежность

- primary и replicas на каждом шарде;
- failover;
- health checks;
- таймауты;
- circuit breaker;
- retry только безопасных операций;
- идемпотентность.

### Наблюдаемость

В метрики и трейсы полезно добавлять:

```text
shard_id
bucket_id
query_name
operation
duration
rows
error_type
```

### Миграции

- версия схемы каждого шарда;
- проверка отставших шардов;
- expand/contract;
- запрет запуска несовместимого кода.

### Резервное копирование

Нужно продумать:

- backup каждого шарда;
- backup routing table;
- WAL archiving;
- восстановление на согласованную временную точку;
- регулярные restore-тесты.

### Глобальные задачи

Отдельно проектируются:

- аналитика;
- поиск без shard key;
- глобальная сортировка;
- уникальные ограничения;
- фоновые обходы всех шардов;
- retention;
- перебалансировка;
- перенос отдельных клиентов.

### Тестирование

Минимальный набор:

- один пользователь всегда попадает на один шард;
- связанные сущности попадают туда же;
- запрос не уходит на неправильный шард;
- частичное падение корректно обрабатывается;
- миграции применяются ко всем шардам;
- scatter-gather обрабатывает ошибки;
- повторная запись идемпотентна;
- routing table обновляется безопасно.

[К оглавлению](#contents)

---

<a id="summary"></a>
## 31. Итог

Шардирование — это разделение строк одной логической базы между несколькими физическими БД.

```text
                    shard key
                        │
                        ▼
Request → Shard Router / Application
              │
      ┌───────┼────────┐
      ▼       ▼        ▼
   shard-0 shard-1  shard-2
```

Основные правила:

1. Шарды хранят разные части данных, а не полные копии.
2. Для выбора шарда нужен shard key.
3. Связанные таблицы желательно шардировать по одному ключу.
4. Связанные строки желательно размещать на одном шарде.
5. В дочерние таблицы shard key часто добавляют явно.
6. Маленькие справочники можно копировать на все шарды.
7. Внешние ключи и транзакции проще сохранять внутри одного шарда.
8. Запрос без shard key часто превращается в scatter-gather.
9. Простое `% shard_count` плохо подходит для масштабирования.
10. Для production удобнее виртуальные бакеты и routing table.
11. Шардирование усложняет JOIN, транзакции, миграции, backup и аналитику.
12. Его применяют, когда более простых способов масштабирования недостаточно.

Главный архитектурный принцип:

> Одна бизнес-операция, один JOIN и одна транзакция по возможности должны выполняться внутри одного шарда.

[К оглавлению](#contents)