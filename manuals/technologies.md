# Технологии

# Содержание

- [Kafka](#kafka)
  - [Абстрактно про устройство](#abstractno-pro-ustroistvo)
  - [Acknowledgements (acks)](#kafka-acks)
- [Гарантии доставки](#delivery-guarantees)
- [Сетевое взаимодействие](#networking)

<a id="kafka"></a>

# Kafka

**Kafka** - распределенный отказоустойчивый append-only журнал событий (брокер сообщений).

Kafka не удаляет сообщение сразу после чтения consumer’ом. Сообщения хранятся до retention policy: по времени, по размеру лога или по compacted policy, если topic настроен как compacted.

<a id="abstractno-pro-ustroistvo"></a>

# Абстрактно про устройство

Kafka cluster состоит из Kafka Nodes

```
Kafka cluster
├── Kafka node 1
├── Kafka node 2
└── Kafka node 3
```

Kafka node - это один запущенный процесс Kafka на одном сервере/контейнере/машине.

Обычно когда говорят, одна Kafka, имеют ввиду один Kafka broker / Kafka node.

Одна Kafka node может выполнять разные роли.

Они могут быть:

- broker
- controller

## **Топик**

**Топик** - логическая сущность, которая представляет из себя именованный поток данных. То есть логическая категория сообщений, разделенная на partitions.

## **Партиция**

**Партиция** - некоторая часть этого потока данных (топика).

Распределение по партициям в рамках одного топика происходит примерно равномерно по hash ключа сообщения (если он есть, подробнее далее).

То есть партиция - отдельный упорядоченный append-only log внутри топика.

Например, у меня есть топик `test.tasks`, я сконфигурировал так, что у меня 3 партиции.

Тогда УСЛОВНО если я буду писать в топик, кафка будет брать партицию номер `hash(key) % 3` и добавлять сообщение в нее.

Порядок сообщений гарантирован только внутри партиции.

Между партициями одного topic порядок не гарантирован.

## **Broker**

**Broker** - роль, которая работает с данными.

Broker - рабочий сервер Kafka, который хранит данные и обслуживает клиентов.

То есть broker:

- хранит партиции на диске
- принимает produce-запросы от producer'ов
- отдает сообщения consumer'ам
- реплицирует данные между leader/follower replicas
- обслуживает metadata-запросы клиентов
- участвует в consumer group coordination

## **Controller**

**Controller** - это управляющая роль Kafka-кластера. Он отвечает за metadata и управление состоянием кластера.

Controller выполняет примерно следующие задачи:

- следит, какие broker'ы живы;
- хранит/обновляет metadata кластера;
- знает, какие topic'и существуют;
- знает, где находятся partitions и replicas;
- назначает leader replicas для partitions;
- переизбирает leader, если broker умер;
- обрабатывает создание/удаление topic'ов;
- распространяет изменения metadata по broker'ам;

При конфигурировании мы указываем для каждой ноды ее роль:

```docker
# docker-compose.yml
services:
  kafka:
    image: apache/kafka:4.1.0
    container_name: kafka-prac
    ports:
      # Для Go-приложения, которое запускается на твоей машине
      - "9092:9092"
    environment:
      # KRaft mode: один node одновременно broker и controller
      KAFKA_NODE_ID: 1
      KAFKA_PROCESS_ROLES: broker,controller

      # Kafka слушает три listener'а:
      # INTERNAL   — для других контейнеров внутри docker compose
      # EXTERNAL   — для приложений с твоей машины
      # CONTROLLER — внутренний listener для KRaft controller
      KAFKA_LISTENERS: INTERNAL://:29092,EXTERNAL://:9092,CONTROLLER://:9093

      # Kafka "рекламирует" клиентам адреса подключения.
      # Контейнерам говорит: ходите на kafka:29092
      # Хосту говорит: ходите на localhost:9092
      KAFKA_ADVERTISED_LISTENERS: INTERNAL://kafka:29092,EXTERNAL://localhost:9092

      # Какие протоколы используются для listener'ов
      KAFKA_LISTENER_SECURITY_PROTOCOL_MAP: INTERNAL:PLAINTEXT,EXTERNAL:PLAINTEXT,CONTROLLER:PLAINTEXT

      # Listener для общения broker'ов между собой.
      # У нас один broker, но Kafka все равно требует это указать.
      KAFKA_INTER_BROKER_LISTENER_NAME: INTERNAL

      # Listener для KRaft controller
      KAFKA_CONTROLLER_LISTENER_NAMES: CONTROLLER
      KAFKA_CONTROLLER_QUORUM_VOTERS: 1@kafka:9093

      # Для одного broker replication factor должен быть 1
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1
      KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR: 1
      KAFKA_TRANSACTION_STATE_LOG_MIN_ISR: 1

      # Чтобы consumer group быстрее стартовала локально
      KAFKA_GROUP_INITIAL_REBALANCE_DELAY_MS: 0

      # Для новых topic'ов по умолчанию
      KAFKA_NUM_PARTITIONS: 3

      # Для обучения удобно
      KAFKA_AUTO_CREATE_TOPICS_ENABLE: "true"
```

При этом если указать, что каждая node и broker, и controller, то это значит, что каждая node умеет быть controller, но не значит, что каждая из них сама по себе независимо управляет своим миром.

Они вместе будут образовывать controller quorum.

То есть controllers будут договариваться между собой, кто сейчас активный controller/leader controller.

То есть они будут работать как единая управляющая группа кластера.

## **Consumer group**

**Consumer group**- это группа консьюмеров, которые слушают один и тот же топик.

Например, у меня есть топик `events`, есть два отдельных сервиса, которые должны по своему обрабатывать события из этого топика.

Например, есть `audit_log_svc` и `event_svc`.

Допустим у меня есть в `audit_log_svc` своя `audit_log_consumer_group`, в которой параллельно консьюмеры этой группы слушают топик `events` и параллельно его обрабатывают и пишут аудит логи.

В то же время в `event_svc` своя `event_consumer_group`, в которой параллельно консьюмеры этой группы слушают топик `events` и параллельно его обрабатывают и генерируют события для пользователей.

Это две разные consumer groups, но они слушают общий топик `events`.

Consumer group может быть не отдельным сервисом/приложением с отдельным сетевым подключением и тд. Это просто группа консьюмеров со своим group_id:

```
audit consumer:
  topic = events
  group_id = "audit-group"

notification consumer:
  topic = events
  group_id = "notification-group"

analytics consumer:
  topic = events
  group_id = "analytics-group"
```

Одно и то же сообщение из топика будет прочитано каждой группой независимо.

При этом если у нас две consumer groups с одинаковым group_id, то kafka считает, что это экземпляры одной и той же группы. То есть одно сообщение будет обработано только одной из групп.

- Таким образом:
  Одинаковый group_id = совместно делят partitions.

- Разный group_id = независимо читают один и тот же topic.

## **Consumer group coordinator**

**Consumer group coordinator** - это дополнительная обязанность, которую выполняет broker.

Выполняет следующие задачи:

- следит за consumer'ами в группе;
- принимает heartbeat;
- запускает rebalance;
- помогает распределять partitions между - consumer'ами;
- управляет commit'ами и состоянием consumer group;

Итоговая примерная схема кластера такая:

```
kafka cluster
├── Kafka node 1 (broker, controller)
|   ├── topic1 partition 0 leader
|   ├── topic1 partition 1 follower
|   ├── topic1 partition 2 follower
|   ├── topic2 partition 0 follower
|   ├── topic2 partition 1 leader
|   ├── topic2 partition 2 follower
|   ├── topic3 partition 0 follower
|   ├── topic3 partition 1 follower
|   └── topic3 partition 2 leader
|
├── Kafka node 2 (broker, controller)
|   ├── topic1 partition 0 follower
|   ├── topic1 partition 1 leader
|   ├── topic1 partition 2 follower
|   ├── topic2 partition 0 follower
|   ├── topic2 partition 1 follower
|   ├── topic2 partition 2 leader
|   ├── topic3 partition 0 leader
|   ├── topic3 partition 1 follower
|   └── topic3 partition 2 follower
|
└── Kafka node 3 (broker, controller)
    ├── topic1 partition 0 follower
    ├── topic1 partition 1 follower
    ├── topic1 partition 2 leader
    ├── topic2 partition 0 leader
    ├── topic2 partition 1 follower
    ├── topic2 partition 2 follower
    ├── topic3 partition 0 follower
    ├── topic3 partition 1 leader
    └── topic3 partition 2 follower
```

Это кластер, для которого мы создали три топика и при создании каждого топика указали, что для него нужно иметь 3 partitions и replication factor равный 3. То есть для каждого топика будет создано 3 partitions, каждая из этих partitions будет иметь по три экземпляра (1 leader + 2 follower).

<a id="kafka-acks"></a>

# Acknowledgements (acks)

Короткий ответ: подтверждения есть и у producer, и у consumer, но это **два разных механизма с разным смыслом**.

- producer получает подтверждение того, что Kafka приняла запись;
- consumer сохраняет позицию, до которой его consumer group считает сообщения пройденными.

## **Producer acks**

Producer отправляет сообщение leader replica нужной partition. Параметр `acks` определяет, скольких подтверждений producer ждет перед тем, как считать отправку успешной.

### `acks=0`

Producer не ждет ответа от broker'а.

- минимальная задержка;
- producer может не узнать даже о том, что broker не принял сообщение;
- сообщение может потеряться.

### `acks=1`

Leader записывает сообщение в свой log и подтверждает запись, не дожидаясь follower replicas.

- надежнее, чем `acks=0`;
- быстрее, чем ожидание всех синхронных replicas;
- сообщение может потеряться, если leader упадет после подтверждения, но до репликации, а новым leader станет replica без этого сообщения.

### `acks=all` (то же самое, что `acks=-1`)

Leader подтверждает запись только после того, как ее получили все replicas, которые на данный момент входят в ISR (in-sync replicas).

ISR - набор replicas, которые достаточно синхронизированы с leader. Это не обязательно все replicas, заданные через replication factor.

`acks=all` само по себе не означает, что запись обязательно оказалась на нескольких broker'ах. За минимальное количество синхронных replicas, необходимое для успешной записи, отвечает настройка topic/broker `min.insync.replicas`.

Например:

```text
replication.factor = 3
min.insync.replicas = 2
producer acks = all
```

В такой конфигурации запись успешна, только если ее могут подтвердить как минимум две in-sync replicas. Если доступна только одна, Kafka отклонит запись вместо того, чтобы принять ее с пониженной надежностью.

На практике для надежной доставки обычно используют `acks=all`, подходящий `min.insync.replicas` и включенную идемпотентность producer'а. `acks` защищает от части потерь, но сам по себе не исключает дубликаты при retry.

## **Consumer ack: commit offset**

На уровне Kafka consumer обычно не отправляет ack для каждого сообщения в том же смысле, что producer. Он делает **commit offset** для своей consumer group. Некоторые клиентские библиотеки называют операцию, после которой выполняется commit, словом `ack`.

У каждого сообщения внутри partition есть offset:

```text
partition 0: [offset 0] [offset 1] [offset 2] [offset 3]
```

Если consumer успешно обработал сообщение с offset `2`, он обычно commit'ит offset `3`. Committed offset означает: «при следующем запуске продолжить чтение отсюда», а не «последним обработан offset 3».

Kafka хранит committed offsets отдельно для каждой комбинации:

```text
consumer group + topic + partition
```

Поэтому разные consumer groups независимо читают один topic и имеют разные позиции.

Commit бывает:

- **автоматическим** - клиент периодически commit'ит offsets сам;
- **ручным** - приложение выполняет commit после выбранного этапа обработки.

Важно, в какой момент выполняется commit:

- commit **до обработки**: при падении после commit сообщение может остаться необработанным; это приближает семантику к at-most-once;
- commit **после успешной обработки**: при падении после обработки, но до commit сообщение будет прочитано повторно; это at-least-once, поэтому обработчик желательно делать идемпотентным.

То есть consumer commit не удаляет сообщение из Kafka. Он только сохраняет позицию consumer group. Само сообщение продолжает храниться согласно retention policy.

## **Как связаны эти подтверждения**

Обычный жизненный цикл выглядит так:

```text
producer
  -> отправляет сообщение
  -> Kafka записывает и отвечает согласно producer acks

consumer
  -> читает уже записанное сообщение
  -> обрабатывает его
  -> commit'ит следующий offset для своей consumer group
```

Producer ack отвечает на вопрос: **«насколько надежно Kafka приняла сообщение?»**

Consumer commit отвечает на вопрос: **«с какого места этой consumer group продолжать чтение?»**

## **Состояние partition replica**

**Состояние partition replica** - это состояние каждого экземпляра partition в топике.

Экземпляр партиции может быть в состоянии follower или leader.

Для каждой партиции только одна из реплик может быть leader, остальные follower.

Запись продюсерами делается только в leader партиции.

Если брокер, на котором находится leader партиция отвалился, то controller делает одну из follower партиции на другом broker'е leader партицией.

## **Replication factor**

**Replication factor**- это количество всех экземпляров partition.

То есть если replication factor = 3, то будет 1 leader + 2 follower.

## **Чтение и запись**

Чтение и запись идут через leader партицию. Follower партиции нужны для отказоустойчивости.

Синхронизация данных между leader и follower партициями происходит асинхронно и какие-то из follower могут отставать от leader.

Далее про ISR.

**ВАЖНО!!!**

Внутри одной consumer group одна partition назначается только одному consumer'у одновременно.

Допустим, топик `events` имеет 3 партиции

_Пример 1:_

В группе один consumer

```
audit-group
└── audit-consumer-1
```

Тогда один consumer будет читать все 3 партиции:

```
audit-consumer-1
├── events partition 0
├── events partition 1
└── events partition 2
```

То есть группа читает весь топик.

_Пример 2:_

В группе 3 consumer'а

```
audit-group
├── audit-consumer-1
├── audit-consumer-2
└── audit-consumer-3
```

Kafka распределит партиции между ними:

```
audit-group
├── audit-consumer-1 -> events partition 0
├── audit-consumer-2 -> events partition 1
└── audit-consumer-3 -> events partition 2
```

Группа все равно читает весь топик параллельно.

_Пример 3:_

В группе 5 consumer'ов

```
audit-group
├── consumer-1 -> events partition 0
├── consumer-2 -> events partition 1
├── consumer-3 -> events partition 2
├── consumer-4 -> idle
└── consumer-5 -> idle
```

Лишние consumer'ы простаивают, потому что partitions всего 3.

## **Конфигурирование количества partitions/replicas**

Конфигурирование происходит при создании топика.

Количество брокеров само по себе не задает количество партиций и реплик.

_Пример 1_

Например, у меня будет 3 brokers, топик `events` и у него будет 1 partitions и репликация равна 1.

Тогда схема будет такая:

```
kafka cluster
├── Kafka node 1 (broker, controller)
|   └── events partition 0 leader
|
├── Kafka node 2 (broker, controller)
|   └── empty
|
└── Kafka node 3 (broker, controller)
    └── empty
```

_Пример 2_

Например, у меня будет 3 brokers, топик `events` и у него будет 2 partitions и репликация равна 1.

Тогда схема будет такая:

```
kafka cluster
├── Kafka node 1 (broker, controller)
|   └── events partition 0 leader
|
├── Kafka node 2 (broker, controller)
|   └── events partition 1 leader
|
└── Kafka node 3 (broker, controller)
    └── empty
```

_Пример 2_

Например, у меня будет 3 brokers, топик `events` и у него будет 2 partitions и репликация равна 2.

Тогда схема будет такая:

```
kafka cluster
├── Kafka node 1 (broker, controller)
|   ├── events partition 0 follower
|   └── events partition 1 leader
|
├── Kafka node 2 (broker, controller)
|   └── events partition 0 leader
|
└── Kafka node 3 (broker, controller)
    └── events partition 1 follower
```

Или такая (там уже как кафка решит разложить):

```
kafka cluster
├── Kafka node 1 (broker, controller)
|   ├── events partition 0 follower
|   └── events partition 1 leader
|
├── Kafka node 2 (broker, controller)
|   ├── events partition 0 leader
|   └── events partition 1 follower
|
└── Kafka node 3 (broker, controller)
    └── empty
```

_Пример 3_

Например, у меня будет 3 brokers, топик `events` и у него будет 4 partitions и репликация равна 2.

Тогда схема будет такая:

```
kafka cluster
├── Kafka node 1 (broker, controller)
|   ├── events partition 0 follower
|   ├── events partition 1 leader
|   └── events partition 2 follower
|
├── Kafka node 2 (broker, controller)
|   ├── events partition 0 leader
|   ├── events partition 3 leader
|   └── events partition 2 leader
|
└── Kafka node 3 (broker, controller)
    ├── events partition 1 follower
    └── events partition 3 follower
```

Или другая (там уже как кафка решит разложить)

## **ISR**

**ISR** - in-sync replicas.

Это реплики партиций, которые достаточно актуальны относительно leader.

При падении leader новый leader будет из ISR.

Реплики синхронизируются, но не мгновенно.

Поэтому ISR можеть быть не все реплики, а только те, которые достаточно актуальны.

Например, пусть у нас есть топик `events` и у нас есть 1 partitions и репликация равна 3. Допустим у меня отваливается брокер с leader партицией, но вторая реплика актуальна, а третья - нет. Тогда ISR только вторая реплика, она и станет leader.

## **Offset**

**Offset** - это номер сообщения внутри конкретной partition.

```
topic: events

partition 0:
  offset 0: msg-A
  offset 1: msg-B
  offset 2: msg-C
  offset 3: msg-D
  offset 4: msg-E
```

Offset уникален только внутри partition.

То есть:

```
partition 0 offset 0
partition 1 offset 0
partition 2 offset 0
```

Это три разных сообщения.

Для каждой consumer group свой offset.

То есть если например у меня два сервиса (consumer group) читают топик `events`, то у каждого сервиса будет свой offset.

Kafka хранит offset отдельно для каждой комбинации:

```
consumer group + topic + partition
```

Например:

```
audit-group + events + partition 0 -> offset 100
audit-group + events + partition 1 -> offset 70
audit-group + events + partition 2 -> offset 90

notification-group + events + partition 0 -> offset 20
notification-group + events + partition 1 -> offset 15
notification-group + events + partition 2 -> offset 18
```

Тут:

- audit consumer 1 прочитал до 100 сообщения партицию 0
- audit consumer 2 прочитал до 70 сообщения партицию 1
- audit consumer 3 прочитал до 90 сообщения партицию 2
- notification consumer 1 прочитал до 20 сообщения партицию 0
- notification consumer 2 прочитал до 15 сообщения партицию 1
- notification consumer 3 прочитал до 18 сообщения партицию 2

**Commited offset** - это offset, который consumer group сохранила в Kafka как свою текущую позицию чтения.

Committed offset означает следующий offset, с которого нужно продолжить чтение.

То есть если сonsumer group обработала:

```
offset 0
offset 1
offset 2
```

То она коммитит:

```
committed offset 3
```

Это значит, что она обработала сообщения до offset 2 включительно и следующий раз начнет читать с offset 3.

Kafka хранит committed offset во внутреннем топике \_\_consumer_offsets.

Если упращенно, то там лежит информация:
| group_id | topic | partition | committed_offset |
| ------------ | ------ | --------- | ---------------- |
| audit-group | events | 0 | 5 |
| audit-group | events | 1 | 8 |
| audit-group | events | 2 | 3 |
| notify-group | events | 0 | 2 |
| notify-group | events | 1 | 4 |
| notify-group | events | 2 | 1 |

<a id="delivery-guarantees"></a>

# Гарантии доставки

Гарантия доставки описывает, сколько раз сообщение или запрос может привести к обработке:

- **at-most-once** - не более одного раза: возможна потеря, но нет повторной обработки;
- **at-least-once** - не менее одного раза: потерю стараются исключить, но возможны повторы;
- **exactly-once** - ровно один раз: эффект операции возникает один раз, без потерь и дублей.

Важно уточнять, гарантию **чего именно** мы обсуждаем:

- сообщение записано в broker;
- сообщение доставлено consumer'у;
- обработчик был вызван;
- транзакция записала данные в БД;
- внешний бизнес-эффект произошел ровно один раз.

Это разные утверждения. Например, Kafka может ровно один раз записать результат Kafka-транзакции, но не способна сама сделать ровно один банковский перевод во внешней системе.

## **Почему нельзя просто узнать, выполнился ли запрос**

Пусть `Order Service` вызывает `Payment Service`:

```text
Order Service -> Payment Service: charge(order-42, 1000)
Payment Service -> Order Service: 200 OK
```

Ответ может потеряться:

```text
Order Service -> Payment Service: charge(order-42, 1000)
Payment Service: деньги списаны
Payment Service --X--> Order Service: 200 OK потерялся
```

После timeout отправитель не знает, что произошло:

1. запрос не дошел, поэтому операция не выполнялась;
2. запрос дошел, но обработчик упал до записи;
3. операция выполнилась, но ответ не дошел.

Это называется **неоднозначным результатом**. Без дополнительного протокола retry может быть необходим в первых двух случаях и опасен в третьем.

## **At-most-once**

At-most-once означает: обработка произойдет ноль или один раз.

Простейший способ получить эту семантику - отправить один раз и не повторять при ошибке:

```text
Order Service
  -> один HTTP-запрос charge
  -> timeout
  -> retry отсутствует
```

Если запрос или ответ потерялся, `Order Service` не повторяет вызов. Двойного списания из-за retry не будет, но заказ может остаться неоплаченным.

Для Kafka похожая ситуация возникает, если consumer commit'ит offset **до** обработки:

```text
consumer получает message
  -> commit offset
  -> обрабатывает message
```

Если процесс упадет между commit и обработкой, после перезапуска группа начнет со следующего offset. Сообщение останется в Kafka, но эта consumer group его повторно не обработает.

At-most-once подходит, когда потеря допустимее дубля, например для части неважной телеметрии. Для платежей, заказов и изменения остатков обычно недостаточно надежен.

## **At-least-once**

At-least-once означает: система повторяет попытки, пока сообщение не будет успешно обработано. Обработка произойдет один или несколько раз.

Улучшим исходную архитектуру:

```text
Order Service
  -> вызывает Payment Service
  -> при timeout/временной ошибке повторяет запрос
  -> использует exponential backoff и ограничение попыток
```

Retry уменьшает вероятность потери, но создает возможность дубля:

```text
attempt 1 -> деньги списаны -> ответ потерялся
attempt 2 -> деньги списаны повторно
```

Для Kafka consumer обычно получает at-least-once так:

```text
consumer получает message
  -> выполняет бизнес-операцию
  -> commit offset только после успеха
```

Если consumer упал после бизнес-операции, но до commit, Kafka отдаст сообщение повторно. Поэтому «commit после обработки» предотвращает потерю, но не предотвращает повторный эффект.

### Retry и backoff

Повторять следует только временные ошибки:

- timeout и временный сетевой сбой;
- `429 Too Many Requests`;
- `502`, `503`, `504`;
- временная недоступность broker'а или БД.

Обычно не следует автоматически повторять постоянные ошибки, например невалидный запрос с `400`.

Exponential backoff увеличивает паузу между попытками:

```text
100 ms -> 200 ms -> 400 ms -> 800 ms -> ...
```

К задержке добавляют случайный **jitter**, чтобы тысячи клиентов не повторили запрос одновременно. Также нужны timeout, максимальное количество попыток и dead-letter queue либо ручная обработка неисправимых сообщений.

Retry обеспечивает повторные попытки, а не exactly-once.

## **Exactly-once и exactly-once effect**

Строгое end-to-end exactly-once между независимыми сервисами, БД и внешними системами получить сложно. Сеть может оборваться в любой момент, а общая атомарная транзакция обычно отсутствует.

На практике строят **effectively-once / exactly-once effect**:

```text
доставка: at-least-once
+ обработка: идемпотентная
= наблюдаемый бизнес-эффект один раз
```

Если дубль доставили три раза, но все три попытки приводят к одному итоговому состоянию, для бизнеса операция выглядит выполненной ровно один раз.

### Идемпотентность

Идемпотентная операция при повторном выполнении с теми же входными данными не меняет результат после первого успеха.

`SET balance = 100` может быть идемпотентным, а `balance = balance - 100` - нет.

Для неидемпотентной операции используют idempotency key:

```http
POST /payments
Idempotency-Key: order-42-payment
```

`Payment Service` хранит ключ и результат операции:

```text
idempotency_key        status      payment_id
order-42-payment       succeeded   pay-901
```

Повторный запрос с тем же ключом не списывает деньги снова, а возвращает сохраненный результат. Проверка ключа и запись платежа должны происходить в **одной локальной транзакции**. На поле ключа нужен `UNIQUE` constraint: одной проверки `SELECT`, выполняемой отдельно от `INSERT`, недостаточно из-за concurrent requests.

Idempotency key должен обозначать одну логическую операцию. Новый осознанный платеж получает новый ключ.

### Transactional outbox

Рассмотрим dual write в `Order Service`:

```text
1. INSERT order в PostgreSQL
2. publish OrderCreated в Kafka
```

Между шагами процесс может упасть. Тогда заказ уже создан, а событие потеряно. Если переставить шаги местами, событие может уйти, а транзакция заказа - откатиться.

Transactional outbox записывает бизнес-данные и событие в одну БД-транзакцию:

```sql
BEGIN;

INSERT INTO orders (id, status)
VALUES ('order-42', 'created');

INSERT INTO outbox (event_id, event_type, aggregate_id, payload)
VALUES ('event-123', 'OrderCreated', 'order-42', '{...}');

COMMIT;
```

Отдельный relay читает outbox и публикует события в Kafka:

```text
Order Service transaction -> orders + outbox
Outbox Relay -> Kafka -> Payment Service
```

Relay может упасть после publish, но до отметки `sent`, поэтому событие может быть опубликовано повторно. Outbox предотвращает потерю между БД и broker'ом, но сам по себе **не устраняет дубли**.

Relay можно реализовать polling'ом таблицы или CDC-инструментом, который читает transaction log БД.

### Inbox / дедупликация у consumer'а

`Payment Service` использует `event_id` как idempotency key. Полученное событие и бизнес-изменение фиксируются атомарно:

```sql
BEGIN;

WITH accepted_event AS (
    INSERT INTO inbox (event_id)
    VALUES ('event-123')
    ON CONFLICT DO NOTHING
    RETURNING event_id
)
INSERT INTO payments (event_id, order_id, amount)
SELECT event_id, 'order-42', 1000
FROM accepted_event;

COMMIT;
```

На `inbox.event_id` нужен `UNIQUE` constraint. Offset Kafka commit'ится после успешного commit транзакции БД. При повторной доставке запись в inbox показывает, что эффект уже применен.

Если consumer вызывает еще одну внешнюю систему, локальная inbox-таблица не может атомарно управлять той системой. Та система тоже должна принимать idempotency key либо участвовать в другом надежном протоколе.

## **Эволюция взаимодействия двух сервисов**

### Шаг 1: at-most-once

```text
Order Service -> один раз вызывает Payment Service
timeout -> ошибка без retry
```

- дубль из-за повторной отправки отсутствует;
- запрос может потеряться;
- неизвестный результат приходится разбирать вручную.

### Шаг 2: at-least-once

```text
Order Service -> retry с exponential backoff + jitter
Payment Service -> может получить один запрос несколько раз
```

- временная ошибка больше не приводит сразу к потере;
- возможна повторная обработка;
- нужны лимиты retry, метрики и DLQ/ручное восстановление.

### Шаг 3: effectively-once

```text
Order Service DB transaction
  -> orders
  -> outbox(event_id)

Outbox Relay
  -> publish with retry/backoff

Kafka
  -> at-least-once delivery

Payment Service DB transaction
  -> inbox(event_id UNIQUE)
  -> payment/business changes

consumer
  -> commit offset after DB commit
```

- outbox не дает потерять событие при dual write;
- retry/backoff доводит временно неуспешную доставку;
- стабильный `event_id` связывает все повторы одной операции;
- inbox/idempotency key подавляет повторный эффект;
- локальные транзакции атомарно связывают маркер дедупликации с изменением данных.

Это обычно называют exactly-once processing или effectively-once, но точнее говорить: **at-least-once delivery с exactly-once business effect в обозначенной границе**.

## **Краткая таблица**

| Гарантия | Потеря возможна | Повтор возможен | Типичная реализация |
| --- | --- | --- | --- |
| at-most-once | да | нет | одна попытка или commit до обработки |
| at-least-once | стараемся исключить | да | retry и commit после обработки |
| effectively-once | стараемся исключить | доставка может повторяться, эффект нет | at-least-once + idempotency + транзакционная дедупликация |
| строгая exactly-once | нет в заданной границе | нет в заданной границе | единая транзакционная система или специальный протокол с четкими ограничениями |

<a id="networking"></a>

# Сетевое взаимодействие

Сетевые технологии удобно рассматривать слоями:

```text
REST / RPC             архитектурный стиль и модель API
HTTP / gRPC            прикладные протоколы
TLS                    шифрование и аутентификация
TCP / UDP / QUIC       транспорт
IP                     доставка между сетями
Ethernet / Wi-Fi       передача в локальном сегменте
```

Каждый слой добавляет свой header к данным верхнего слоя. При отправке HTTP по TCP примерно происходит инкапсуляция:

```text
HTTP message
-> TCP segment
-> IP packet
-> Ethernet frame
-> электрический/радио/оптический сигнал
```

На принимающей стороне headers разбираются в обратном порядке.

## **IP и прохождение пакета**

IP отвечает за адресацию и доставку packet'ов через несколько сетей. Router смотрит на destination IP, выбирает следующий hop и пересылает packet дальше.

IP работает по принципу best effort:

- packet может потеряться;
- прийти дважды;
- прийти не по порядку;
- задержаться;
- для IPv4 быть фрагментированным при превышении MTU.

MTU - максимальный размер frame payload на участке сети; для обычного Ethernet часто 1500 bytes. TCP обычно подбирает MSS так, чтобы TCP segment вместе с IP header помещался в MTU. Современные приложения стараются избегать IP fragmentation.

Пакеты теряются из-за переполнения очередей router'ов, ошибок канала, Wi-Fi interference, обрыва маршрута, перегруженного receiver'а, истечения TTL и других причин. IP не восстанавливает потерю: это делает транспортный или прикладной протокол, если ему требуется надежность.

## **Порты и sockets**

IP address указывает host/interface, а port - приложение или endpoint внутри host.

Соединение TCP обычно идентифицируется tuple:

```text
protocol + source IP + source port + destination IP + destination port
```

Socket - интерфейс ОС, через который приложение отправляет и принимает данные. Server делает `bind` к адресу/порту и ожидает трафик; client обычно получает временный ephemeral port.

## **UDP**

UDP - простой connectionless транспортный протокол, передающий отдельные datagrams.

UDP header содержит в основном:

- source port;
- destination port;
- length;
- checksum.

Перед отправкой нет handshake и постоянного соединения. Приложение передает datagram ОС, UDP добавляет header, IP доставляет packet по возможности.

UDP не гарантирует:

- доставку;
- порядок;
- отсутствие дублей;
- автоматический retry;
- flow control;
- congestion control на уровне самого UDP.

Граница сообщений сохраняется: две отправленные datagrams при получении не склеиваются в один byte stream. Но datagram может потеряться целиком, а слишком крупная - потребовать IP fragmentation и стать более уязвимой к потере.

UDP выбирают, когда важны низкая задержка, multicast/broadcast или приложение хочет само управлять надежностью: DNS, часть real-time audio/video, игры, VPN и QUIC. «UDP быстрее» не означает «UDP всегда лучше»: необходимую надежность приложению иногда приходится реализовывать самостоятельно.

## **TCP**

TCP - connection-oriented надежный упорядоченный **byte stream** поверх IP.

TCP дает приложению:

- байты без пропусков и в правильном порядке;
- обнаружение потерь и retransmission;
- flow control, чтобы не переполнить receiver;
- congestion control, чтобы не перегрузить сеть;
- full-duplex передачу в обе стороны.

TCP не сохраняет границы сообщений. Если приложение дважды вызвало `write` для `hello` и `world`, receiver может прочитать `helloworld`, несколько маленьких частей или другое разбиение. Прикладной протокол сам задает framing, например длиной сообщения, delimiter'ом или HTTP headers.

### TCP three-way handshake

Упрощенное установление соединения:

```text
client -> server: SYN, seq=x
server -> client: SYN-ACK, seq=y, ack=x+1
client -> server: ACK, ack=y+1
```

Handshake подтверждает доступность обеих сторон, синхронизирует начальные sequence numbers и согласовывает TCP options, например MSS и window scaling. После него начинается передача application data.

TLS handshake для HTTPS - отдельный handshake поверх установленного TCP. Поэтому традиционный HTTPS требует сначала TCP handshake, затем TLS handshake. TLS 1.3 уменьшает количество дополнительных round trips, а connection reuse позволяет не повторять handshakes для каждого запроса.

### Сегментация, порядок и retransmission

Приложение пишет byte stream в socket. TCP делит его на segments подходящего размера и нумерует байты sequence numbers. Receiver подтверждает полученные данные ACK'ами.

Если ACK не пришел вовремя или несколько ACK указывают на пропуск, sender повторно передает потерянные данные. Receiver временно буферизует более поздние segments, но приложение получает непрерывный поток по порядку.

Поэтому потеря одного TCP segment может задержать выдачу более поздних данных приложению. Это **head-of-line blocking**.

Flow control использует receive window и защищает receiver. Congestion control меняет скорость отправки на основании признаков перегрузки сети. Эти механизмы решают разные задачи.

### Завершение соединения

Обычное закрытие использует FIN/ACK в каждом направлении, потому что TCP full-duplex и стороны закрывают свои потоки независимо. RST аварийно обрывает соединение.

TCP гарантирует надежный поток только пока endpoints и соединение существуют. Успешный TCP ACK означает, что байты принял TCP stack другой машины, а не то, что приложение завершило бизнес-операцию или записало данные в БД.

## **UDP и TCP: сравнение**

| Свойство | UDP | TCP |
| --- | --- | --- |
| Установка соединения | нет | three-way handshake |
| Модель данных | datagrams | byte stream |
| Доставка и порядок | не гарантируются | гарантируются в рамках соединения |
| Retransmission | нет | есть |
| Flow/congestion control | нет в UDP | есть |
| Цена потери | приложение решает само | задержка из-за retransmission и порядка |

## **Как работает BitTorrent**

BitTorrent - peer-to-peer прикладной протокол распространения файлов. Это не замена TCP или UDP: разные реализации передают peer traffic поверх TCP или uTP, который реализует собственную надежность поверх UDP.

Упрощенно:

1. Файл делится на pieces, а pieces могут передаваться blocks.
2. Metadata содержит cryptographic hashes, позволяющие проверить pieces.
3. Клиент находит peers через tracker, DHT, peer exchange или известный peer.
4. Peers сообщают, какие pieces у них есть.
5. Клиент одновременно скачивает разные pieces у разных peers и раздает уже проверенные pieces другим.
6. Поврежденный piece не проходит hash verification и скачивается снова.

Стратегия rarest-first помогает сначала распространять редкие pieces, а choking/unchoking стимулирует обмен bandwidth. Tracker обычно помогает peers найти друг друга, но не обязан хранить или передавать сам файл. DHT позволяет находить peers без центрального tracker'а.

Torrent не собирает произвольные потерянные IP packets самостоятельно, если работает поверх TCP: retransmission делает TCP. BitTorrent управляет более крупным уровнем - выбором peers и pieces, параллельным скачиванием и проверкой целостности.

## **HTTP**

HTTP - прикладной request/response протокол. Client отправляет request с method, target, headers и иногда body; server возвращает status, headers и body.

HTTP определяет смысл сообщений, а транспорт переносит bytes. Обычно HTTP/1.1 и HTTP/2 работают поверх TCP; HTTP/3 - поверх QUIC/UDP.

HTTP сам по себе stateless: каждый request содержит достаточно контекста для обработки. Состояние сессии добавляют cookies, tokens и server-side storage.

### HTTP/1.0 и HTTP/1.1

HTTP/1.x имеет текстовый формат start line и headers:

```http
GET /orders/42 HTTP/1.1
Host: api.example.com
Accept: application/json
```

Body при этом всегда мог быть бинарным, например JPEG. «Текстовый HTTP/1.1» означает, что управляющая часть протокола сериализована как строки с delimiters, а не то, что по сети передаются не bytes.

HTTP/1.0 обычно создавал отдельное TCP connection для каждого запроса. HTTP/1.1 сделал persistent connections стандартным поведением и добавил обязательный `Host`, chunked transfer encoding и другие возможности.

HTTP/1.1 pipelining позволял отправлять несколько запросов без ожидания, но ответы должны были идти по порядку. На практике browsers чаще открывали несколько TCP connections, потому что медленный первый response блокировал следующие.

### HTTP/2

HTTP/2 сохраняет семантику HTTP: methods, URLs, status codes и headers. Меняется wire format и способ передачи.

Сообщения разбиваются на бинарные frames, объединенные в независимые streams внутри одного TCP connection:

```text
TCP connection
├── stream 1: HEADERS + DATA
├── stream 3: HEADERS + DATA
└── stream 5: HEADERS + DATA
```

Frames разных streams можно multiplex'ить. Медленный HTTP response больше не обязан на уровне HTTP блокировать frames другого response. HTTP/2 также использует HPACK для сжатия headers и поддерживает stream prioritization/flow control.

Почему HTTP/2 называют бинарным, хотя HTTP/1.1 тоже является bytes:

- любой сетевой протокол в итоге передается битами/байтами;
- в HTTP/1.1 структура управляющей части определяется текстовыми строками и separators (`CRLF`, двоеточие и т.д.);
- в HTTP/2 структура определяется бинарными полями фиксированного формата: length, frame type, flags, stream ID;
- «бинарный» здесь описывает framing и parsing, а не физический вид данных в проводе.

HTTP/2 multiplexing не полностью устраняет blocking: все streams идут через один упорядоченный TCP byte stream. Потеря TCP segment может временно задержать данные всех streams на этом connection.

### HTTP/3 и QUIC

HTTP/3 использует QUIC поверх UDP. QUIC реализует шифрование TLS 1.3, надежность, congestion control и несколько независимых streams в user space.

Потеря данных одного QUIC stream не блокирует выдачу данных других streams так, как потеря TCP segment блокирует общее TCP connection. QUIC также быстрее восстанавливает соединение при смене сети, например при переходе телефона с Wi-Fi на mobile network.

UDP здесь не означает, что HTTP/3 ненадежен: надежность реализует QUIC поверх UDP.

## **HTTPS и TLS**

HTTPS - HTTP поверх защищенного TLS transport.

TLS обеспечивает:

- **confidentiality** - трафик зашифрован;
- **integrity** - изменение данных обнаруживается;
- **authentication** - client проверяет server certificate и имя host;
- опционально mutual TLS, когда server также проверяет certificate client'а.

В TLS handshake стороны согласуют версию и cryptographic parameters, server отправляет certificate, client проверяет цепочку доверия и владение ключом, после чего стороны получают session keys.

HTTPS не делает недоверенный server безопасным и не исправляет уязвимости приложения. Оно защищает канал между TLS endpoints. Reverse proxy может завершить TLS, после чего до backend идет отдельное соединение.

## **RPC**

RPC - модель взаимодействия, в которой удаленный вызов оформляется похоже на вызов функции:

```text
CreateOrder(request) -> response
```

RPC framework обычно отвечает за serialization, transport, client/server stubs, deadlines, metadata и отображение ошибок. Но удаленный вызов не равен локальному:

- сеть имеет latency и может быть недоступна;
- результат после timeout может быть неизвестен;
- request/response имеют ограничение размера и стоимость serialization;
- retry может повторить бизнес-операцию;
- обе стороны развиваются и требуют version compatibility.

Поэтому RPC API должны явно использовать deadlines, cancellation, idempotency и продуманную модель ошибок.

## **gRPC**

gRPC - RPC framework, обычно использующий Protocol Buffers как IDL и serialization и HTTP/2 как transport.

В `.proto` описывается контракт:

```proto
service PaymentService {
  rpc Charge(ChargeRequest) returns (ChargeResponse);
}
```

Из контракта генерируются typed client/server stubs. gRPC поддерживает:

- unary request/response;
- server streaming;
- client streaming;
- bidirectional streaming;
- deadlines, cancellation, metadata и status codes.

Плюсы gRPC: строгий контракт, компактный binary format, code generation, streaming и хорошая производительность для service-to-service communication.

Минусы: payload сложнее читать вручную, browser support требует дополнительных решений, public API и debugging через обычные HTTP-инструменты менее просты, а schema evolution требует соблюдать правила совместимости protobuf.

gRPC не гарантирует exactly-once. Deadline или retry оставляет ту же неоднозначность результата, поэтому неидемпотентным методам нужны idempotency keys.

## **REST и RPC**

REST - архитектурный стиль, ориентированный на resources и uniform interface. RPC ориентирован на operations/procedures.

Один и тот же API можно выразить так:

```http
# REST-подобный API
POST /payments
GET /payments/pay-901
DELETE /payments/pay-901

# RPC-подобный HTTP API
POST /payments.create
POST /payments.get
POST /payments.cancel
```

Для REST характерны:

- resource имеет URI;
- стандартный method выражает действие (`GET`, `POST`, `PUT`, `PATCH`, `DELETE`);
- HTTP status codes и headers сохраняют стандартный смысл;
- representations ресурса передаются, например, в JSON;
- requests stateless;
- responses могут использовать HTTP caching.

Для RPC характерны:

- API моделируется набором методов;
- имена отражают бизнес-команды (`Charge`, `ApproveOrder`);
- contract часто описывает точные request/response types;
- transport может быть HTTP, HTTP/2, TCP или другой протокол.

REST и RPC не являются транспортами и не являются противоположностью HTTP. REST часто реализуют поверх HTTP, а RPC тоже может работать поверх HTTP. gRPC - конкретный RPC framework; JSON endpoint с `/orders/approve` также может быть RPC-style API.

### Когда выбирать REST

REST удобен для resource-oriented CRUD API, public APIs, browser clients, стандартного HTTP caching и широкой совместимости инструментов.

Не каждое бизнес-действие естественно выражается CRUD. Команды вроде `approve`, `capture` или `recalculate` иногда понятнее оставить явными actions, чем искусственно маскировать под изменение ресурса.

### Когда выбирать RPC/gRPC

RPC/gRPC удобен для внутренних сервисов со строгими контрактами, большим количеством команд, polyglot code generation, низкими накладными расходами и streaming.

Выбор не обязан быть глобальным: система может предоставлять REST/JSON public API и использовать gRPC между внутренними сервисами.

## **Итоговая связь понятий**

```text
REST или RPC             как спроектирован API
HTTP или gRPC            как оформлены сообщения и вызовы
TLS                      как защищен канал
TCP или QUIC/UDP         как bytes доставляются между процессами
IP                       как packets проходят между сетями
Ethernet/Wi-Fi           как frames проходят текущий физический участок
```

Надежность нижнего слоя не заменяет подтверждение верхнего. TCP ACK подтверждает прием bytes сетевым стеком, HTTP `200` - ответ приложения, Kafka commit - позицию consumer group, а exactly-once business effect требует отдельного проектирования транзакций и идемпотентности.
