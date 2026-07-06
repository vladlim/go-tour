# Технологии

# Содержание

- [Kafka](#kafka)
  - [Абстрактно про устройство](#abstractno-pro-ustroistvo)

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
