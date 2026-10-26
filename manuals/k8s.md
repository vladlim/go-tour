# Kubernetes — верхнеуровневый конспект для собеседования

## Оглавление

1. [Что такое Kubernetes](#what-is-kubernetes)
2. [Зачем нужен Kubernetes](#why-kubernetes)
3. [Кластер, Node и Pod](#cluster-node-pod)
4. [Control Plane](#control-plane)
5. [Worker Node](#worker-node)
6. [Как создаётся кластер](#cluster-bootstrap)
7. [Как проходит kubectl apply](#kubectl-apply-flow)
8. [Декларативная модель и reconciliation](#declarative-model)
9. [Workload-объекты](#workloads)
10. [Deployment, ReplicaSet и Pod](#deployment-replicaset-pod)
11. [Service и сетевой доступ](#service-networking)
12. [Ingress и внешний трафик](#ingress)
13. [Сеть Kubernetes и CNI](#cni)
14. [Хранение данных: Volume, PV, PVC, StorageClass](#storage)
15. [ConfigMap и Secret](#configmap-secret)
16. [Requests и limits](#resources)
17. [Probes](#probes)
18. [Namespaces, ServiceAccount и RBAC](#security-and-isolation)
19. [Stateful-приложения и StatefulSet](#stateful)
20. [Базы данных, брокеры и failover](#databases-and-brokers)
21. [Operator и CRD](#operator-crd)
22. [Самовосстановление и отказы](#self-healing)
23. [Масштабирование](#autoscaling)
24. [Обновление приложения](#rollout)
25. [Kubernetes и Docker Compose](#kubernetes-vs-compose)
26. [Что Kubernetes не делает](#what-kubernetes-does-not-do)
27. [Полная картина работы Kubernetes](#full-picture)
28. [Частые вопросы на собеседовании](#interview-questions)
29. [Краткая шпаргалка](#cheat-sheet)

---

<a id="what-is-kubernetes"></a>

## Что такое Kubernetes

**Kubernetes — система оркестрации контейнеризированных приложений.**

Он управляет:

- запуском контейнеров;
- размещением по серверам;
- количеством реплик;
- сетью;
- конфигурацией;
- storage;
- обновлениями;
- восстановлением после отказов;
- масштабированием.

Главная идея:

```text
Docker запускает контейнеры на одной машине.

Kubernetes управляет контейнерами
на одной или нескольких машинах.
```

Kubernetes — не одна программа на одном сервере. Это набор процессов, работающих на разных машинах и взаимодействующих по сети:

```text
Kubernetes cluster
├── Control Plane
└── Worker Nodes
```

---

<a id="why-kubernetes"></a>

## Зачем нужен Kubernetes

Docker Compose удобен, когда:

- сервер один;
- контейнеров немного;
- не требуется сложная отказоустойчивость;
- не требуется автоматическое масштабирование.

Когда серверов и сервисов становится много, появляются задачи:

- где запустить новый контейнер;
- на каком сервере достаточно CPU и RAM;
- как запустить несколько реплик;
- как заменить упавшую реплику;
- как обновить приложение без простоя;
- как дать сервису постоянный адрес;
- как подключить конфигурацию и secrets;
- как организовать persistent storage;
- как распределять внешний трафик.

Kubernetes решает эти задачи как единая платформа.

---

<a id="cluster-node-pod"></a>

## Кластер, Node и Pod

### Cluster

**Cluster — совокупность машин и компонентов Kubernetes, которые работают как одна система.**

```text
Cluster
├── Control Plane Nodes
└── Worker Nodes
```

### Node

**Node — машина, включённая в Kubernetes-кластер.**

Node может быть:

- физическим сервером;
- виртуальной машиной;
- локальной учебной машиной.

Обычно:

```text
1 Node ≈ 1 сервер или 1 VM
```

У каждой Node свои CPU, RAM, диск, IP, Linux kernel и процессы. Память и CPU между Nodes физически не общие.

### Pod

**Pod — минимальная единица развёртывания в Kubernetes.**

Pod содержит один или несколько контейнеров:

```text
Pod
└── Container
    └── Process
```

Или:

```text
Pod
├── Container app
└── Container sidecar
```

Контейнеры одного Pod:

- всегда находятся на одной Node;
- имеют общий IP;
- используют общее пространство портов;
- могут общаться через `localhost`;
- могут подключать общие volumes;
- создаются и удаляются вместе.

Важно:

```text
Pod ≠ Container

Pod содержит Container.
```

Обычно:

```text
1 Pod = 1 основной контейнер приложения
```

---

<a id="control-plane"></a>

## Control Plane

**Control Plane — управляющая часть Kubernetes-кластера.**

Он решает:

- какие объекты должны существовать;
- сколько Pod нужно;
- на какой Node назначить Pod;
- нужно ли создать замену упавшему Pod;
- нужно ли выполнить rollout;
- соответствует ли реальное состояние желаемому.

Основные компоненты:

```text
Control Plane
├── kube-apiserver
├── etcd
├── kube-scheduler
└── kube-controller-manager
```

### kube-apiserver

Центральная точка входа в Kubernetes.

Через API server общаются:

- `kubectl`;
- kubelet;
- scheduler;
- controllers;
- operators;
- CI/CD;
- внешние инструменты.

```text
kubectl
   ↓ HTTPS
kube-apiserver
```

API server:

- принимает объект;
- валидирует его;
- проверяет аутентификацию;
- проверяет права;
- запускает admission checks;
- сохраняет состояние;
- возвращает ответ.

### etcd

**etcd — распределённое key-value хранилище состояния Kubernetes.**

В нём хранится информация о Nodes, Deployments, Pods, Services, ConfigMaps, Secrets и статусах объектов.

```text
kube-apiserver
      ↓
    etcd
```

etcd — база самого Kubernetes, а не бизнес-база приложения.

### kube-scheduler

Scheduler отвечает на вопрос:

```text
На какой Node разместить новый Pod?
```

Он учитывает:

- requests CPU и RAM;
- labels Nodes;
- affinity и anti-affinity;
- taints и tolerations;
- topology;
- доступность volume;
- специализированное оборудование.

Упрощённо:

```text
Filtering
→ убрать неподходящие Nodes

Scoring
→ выбрать лучшую Node
```

Scheduler не запускает контейнер. Он только записывает:

```text
Pod A назначен на Node 2
```

### kube-controller-manager

Запускает множество контроллеров:

- Deployment Controller;
- ReplicaSet Controller;
- Node Controller;
- Job Controller;
- Namespace Controller;
- другие.

Контроллеры постоянно сравнивают желаемое и реальное состояние.

---

<a id="worker-node"></a>

## Worker Node

**Worker Node — машина, на которой реально запускаются Pods приложения.**

Основные компоненты:

```text
Worker Node
├── kubelet
├── container runtime
├── CNI-компоненты
├── kube-proxy или замена
└── Pods
```

### kubelet

`kubelet` — агент Kubernetes на каждой Node.

Он:

- получает информацию о Pod, назначенных этой Node;
- просит runtime скачать image;
- запускает контейнеры;
- подключает volumes;
- выполняет probes;
- передаёт status в API server;
- следит, чтобы назначенные Pod работали.

Главное:

```text
Control Plane принимает решение.

kubelet на Node выполняет решение.
```

### Container runtime

Runtime непосредственно запускает контейнеры. Обычно это containerd или CRI-O.

```text
kubelet
   ↓ CRI
containerd
   ↓
runc
   ↓
Linux kernel
```

**CRI — Container Runtime Interface**, интерфейс между kubelet и runtime.

Kubernetes не требует установленного Docker Engine. Image можно собрать через Docker, а запустить через containerd.

### kube-proxy

`kube-proxy` традиционно реализует сетевое поведение Service.

Он настраивает правила, которые направляют трафик:

```text
Service
   ↓
Pod A / Pod B / Pod C
```

Часто это не обычный userspace reverse proxy, а сетевые правила Linux. Некоторые CNI-решения могут заменить kube-proxy.

---

<a id="cluster-bootstrap"></a>

## Как создаётся кластер

Нужно различать:

```text
kubeadm  → создаёт или расширяет кластер
kubelet  → работает на каждой Node
kubectl  → клиент управления
containerd → запускает контейнеры
```

### Пример с четырьмя серверами

```text
server-1 → control-plane
server-2 → worker
server-3 → worker
server-4 → worker
```

На все серверы устанавливаются:

- containerd;
- kubelet;
- kubeadm.

`kubectl` не обязательно устанавливать на все машины. Его можно установить на ноутбук, control-plane сервер, CI/CD runner или отдельную admin-машину.

### Создание первой control-plane Node

На `server-1` выполняется:

```bash
kubeadm init
```

Эта команда:

- создаёт control-plane конфигурацию;
- создаёт сертификаты;
- настраивает API server;
- запускает scheduler;
- запускает controller-manager;
- запускает etcd;
- создаёт команду `kubeadm join`.

### Присоединение workers

На других серверах выполняется:

```bash
kubeadm join api.cluster.local:6443 \
  --token ... \
  --discovery-token-ca-cert-hash sha256:...
```

В ней явно указан адрес API server. После этого kubelet worker Node знает, к какому Kubernetes API обращаться.

### Важно

```text
kubeadm init
→ создаёт первую control-plane Node

kubeadm join
→ присоединяет worker или дополнительную control-plane Node

kubectl apply
→ создаёт объекты в уже работающем кластере
```

`kubectl apply` не превращает сервер в control plane.

---

<a id="kubectl-apply-flow"></a>

## Как проходит kubectl apply

Пусть есть файл:

```yaml
apiVersion: apps/v1
kind: Deployment

metadata:
  name: report-service

spec:
  replicas: 3
```

Выполняется:

```bash
kubectl apply -f deployment.yaml
```

Полный путь:

```text
1. kubectl читает YAML.
2. kubectl отправляет запрос в kube-apiserver.
3. API server проверяет объект.
4. Объект сохраняется в etcd.
5. Deployment Controller замечает Deployment.
6. Создаётся ReplicaSet.
7. ReplicaSet Controller создаёт Pods.
8. Scheduler выбирает Node для каждого Pod.
9. В объекте Pod записывается nodeName.
10. kubelet выбранной Node видит новый Pod.
11. kubelet просит container runtime скачать image.
12. CNI настраивает сеть Pod.
13. Подключаются volumes.
14. Runtime запускает контейнер.
15. В контейнере запускается процесс приложения.
16. kubelet отправляет status обратно в API server.
```

Схема:

```text
YAML
 ↓
kubectl
 ↓
API server
 ↓
etcd
 ↓
controllers
 ↓
Pod objects
 ↓
scheduler
 ↓
kubelet
 ↓
container runtime
 ↓
container
 ↓
process
```

---

<a id="declarative-model"></a>

## Декларативная модель и reconciliation

Kubernetes работает декларативно.

Пользователь описывает, **что должно существовать**, а не пошаговые команды.

У каждого объекта есть:

```text
spec   → желаемое состояние
status → фактическое состояние
```

Пример:

```text
spec.replicas = 3
status.readyReplicas = 2
```

Контроллер видит разницу и создаёт ещё один Pod.

Цикл управления:

```text
1. Прочитать desired state.
2. Прочитать actual state.
3. Сравнить.
4. Выполнить изменения.
5. Повторить.
```

Это называется **reconciliation loop**.

---

<a id="workloads"></a>

## Workload-объекты

### Deployment

Для stateless-приложений: backend, frontend, API gateway, взаимозаменяемые workers.

Поддерживает replicas, rolling update, rollback, масштабирование и замену Pod.

### ReplicaSet

Поддерживает нужное количество одинаковых Pod. Обычно создаётся и управляется Deployment.

### StatefulSet

Для приложений, которым нужны стабильные имена Pod, стабильная идентичность, отдельный volume для каждой реплики и управляемый порядок запуска.

Примеры: PostgreSQL, Kafka, ZooKeeper, distributed databases.

### DaemonSet

Запускает Pod на каждой или выбранных Nodes.

Примеры: log agent, monitoring agent, CNI agent, security agent.

### Job

Запускает задачу до успешного завершения.

### CronJob

Создаёт Job по расписанию.

---

<a id="deployment-replicaset-pod"></a>

## Deployment, ReplicaSet и Pod

Основная иерархия stateless-приложения:

```text
Deployment
    ↓ управляет
ReplicaSet
    ↓ поддерживает количество
Pods
    ↓ содержат
Containers
    ↓ запускают
Processes
```

Пример:

```text
Deployment report-service
└── ReplicaSet report-service-7d89f
    ├── Pod report-service-abc
    ├── Pod report-service-def
    └── Pod report-service-ghi
```

Почему не создают Pod напрямую:

- Pod временный;
- при удалении сам не восстановится;
- при обновлении нет rollout;
- нет управления количеством реплик.

Обычно создают Deployment, а он уже управляет остальными уровнями.

---

<a id="service-networking"></a>

## Service и сетевой доступ

Pod эфемерны. При пересоздании Pod может получить новый UID, IP и Node. Нельзя надёжно подключаться к конкретному Pod IP.

### Service

**Service — стабильный сетевой endpoint для группы Pod.**

```text
Service report-service
├── Pod A
├── Pod B
└── Pod C
```

Клиент обращается:

```text
http://report-service:8080
```

Service выбирает Pod через labels и selector.

Если Pod удалился и появился новый, клиент всё равно использует одно имя `report-service`.

### Типы Service

- `ClusterIP` — доступен внутри кластера;
- `NodePort` — открывает порт на каждой Node;
- `LoadBalancer` — запрашивает внешний load balancer;
- `ExternalName` — DNS-отображение на внешний адрес.

---

<a id="ingress"></a>

## Ingress и внешний трафик

Ingress нужен для HTTP/HTTPS-маршрутизации.

```text
api.example.com/auth
→ auth-service

api.example.com/reports
→ report-service
```

Путь запроса:

```text
Internet
   ↓
External Load Balancer
   ↓
Ingress Controller
   ↓
Service
   ↓
Pod
```

Важно:

```text
Ingress
→ Kubernetes-объект с правилами маршрутизации

Ingress Controller
→ реальная программа, исполняющая эти правила
```

Ingress Controller не обязателен для всего Kubernetes. Он нужен преимущественно для внешнего HTTP/HTTPS-доступа.

---

<a id="cni"></a>

## Сеть Kubernetes и CNI

**CNI — Container Network Interface.**

CNI-плагин:

- создаёт сетевой интерфейс;
- выдаёт Pod IP;
- настраивает маршруты;
- обеспечивает связь Pod между Nodes;
- может реализовывать NetworkPolicy.

Примеры: Cilium, Calico, Flannel, Antrea.

### Сетевая модель

```text
Container
   ↓
Pod network namespace
   ↓
Pod IP
   ↓
CNI network
   ↓
другие Pods
```

Pods на разных Nodes общаются через физическую сеть серверов и маршруты CNI.

Внутри одного Pod контейнеры общаются через `localhost`. Между разными Pods `localhost` не работает.

---

<a id="storage"></a>

## Хранение данных: Volume, PV, PVC, StorageClass

Файловая система контейнера временная.

### Volume

Volume подключается внутрь Pod и может быть общим для его контейнеров.

### PersistentVolume

**PV — Kubernetes-объект, представляющий постоянное хранилище.**

За ним может стоять облачный диск, Ceph, NFS, SAN, локальный диск или другое storage.

### PersistentVolumeClaim

**PVC — запрос приложения на storage.**

```text
Pod
 ↓
PVC
 ↓
PV
 ↓
реальный диск
```

### StorageClass

Описывает класс storage: `fast-ssd`, `slow-hdd`, `encrypted`, `replicated`.

Может обеспечивать dynamic provisioning.

Важно:

```text
PV сохраняет данные отдельно от Pod.

Но PV сам по себе не заменяет:
- replication;
- failover;
- backup.
```

---

<a id="configmap-secret"></a>

## ConfigMap и Secret

### ConfigMap

Хранит несекретную конфигурацию: `LOG_LEVEL`, `DATABASE_HOST`, `HTTP_PORT`.

Может подключаться как environment variables, файлы или аргументы.

### Secret

Хранит чувствительные данные: password, token, TLS certificate, registry credentials.

Важно:

```text
Base64 ≠ encryption
```

Для Secrets нужны RBAC, минимальные права и encryption at rest.

---

<a id="resources"></a>

## Requests и limits

```yaml
resources:
  requests:
    cpu: "250m"
    memory: "128Mi"
  limits:
    cpu: "500m"
    memory: "256Mi"
```

### Request

Сколько ресурсов учитывать при размещении Pod. Scheduler использует requests, чтобы понять, помещается ли Pod на Node.

### Limit

Верхняя граница использования.

```text
CPU limit превышен
→ throttling

Memory limit превышен
→ возможен OOM kill
```

Главное:

```text
request
→ планирование и гарантируемая потребность

limit
→ верхняя граница
```

---

<a id="probes"></a>

## Probes

### Startup Probe

Проверяет, успело ли приложение запуститься.

### Liveness Probe

Проверяет, живо ли приложение. При провале kubelet может перезапустить контейнер.

### Readiness Probe

Проверяет, готово ли приложение принимать трафик. Если readiness не проходит, Pod исключается из backend-ов Service.

```text
Процесс работает
База временно недоступна

liveness = true
readiness = false
```

---

<a id="security-and-isolation"></a>

## Namespaces, ServiceAccount и RBAC

### Namespace

Логически группирует ресурсы:

```text
Cluster
├── namespace dev
├── namespace staging
└── namespace prod
```

Namespace помогает разделять окружения, команды, RBAC, quotas и NetworkPolicy, но сам по себе не является полной security boundary.

### ServiceAccount

Идентичность workload внутри Kubernetes.

### RBAC

Определяет:

```text
Кто
может выполнить какое действие
над каким объектом
```

---

<a id="stateful"></a>

## Stateful-приложения и StatefulSet

Stateless Pods взаимозаменяемы:

```text
app-1 = app-2 = app-3
```

Stateful replicas могут иметь разные роли:

```text
postgres-0 → primary
postgres-1 → replica
postgres-2 → replica
```

StatefulSet даёт:

- стабильные имена;
- стабильную сетевую идентичность;
- отдельный PVC для каждой реплики;
- управляемый порядок запуска и удаления.

Но StatefulSet не знает:

- кто primary;
- кто replica;
- кого повышать;
- как избежать split-brain;
- насколько актуальна replica.

StatefulSet предоставляет инфраструктурные гарантии, но не бизнес-логику базы.

---

<a id="databases-and-brokers"></a>

## Базы данных, брокеры и failover

### PostgreSQL

Обычный Service не понимает PostgreSQL-роли. Поэтому обычно нужны разные endpoints:

```text
postgres-rw
→ только current primary

postgres-ro
→ replicas
```

Backend подключается к `postgres-rw`.

Если primary упал:

```text
1. PostgreSQL operator или cluster manager выбирает replica.
2. Replica повышается до primary.
3. Обновляются labels или endpoints.
4. postgres-rw начинает указывать на новый primary.
5. Приложение переподключается по тому же DNS-имени.
```

Существующее TCP-соединение не переносится автоматически. Приложение должно обнаружить ошибку, закрыть соединение, переподключиться и корректно обрабатывать retries.

### Kafka

Клиент подключается к bootstrap Service, получает metadata и затем обращается к нужным brokers. Если leader partition падает, Kafka выбирает нового leader, а клиент обновляет metadata.

```text
Kubernetes
→ Pods, Nodes, Services, volumes

Kafka
→ partitions, replicas, leaders

Operator
→ связывает Kubernetes и Kafka
```

---

<a id="operator-crd"></a>

## Operator и CRD

### CRD

**CustomResourceDefinition — способ добавить новый тип объекта в Kubernetes API.**

### Operator

**Operator — специальный Controller, понимающий конкретное приложение.**

PostgreSQL Operator понимает primary, replicas, WAL, promotion, backup, failover и replication lag.

Operator может создавать Pods, Services, PVC, конфигурацию, backup и failover-механизмы.

---

<a id="self-healing"></a>

## Самовосстановление и отказы

### Упал контейнер

kubelet может перезапустить его.

### Упал Pod Deployment

ReplicaSet создаёт новый Pod.

### Упала Worker Node

Control Plane может создать заменяющие Pods на других Nodes.

### Упал единственный Control Plane

Worker автоматически не становится control plane.

Уже запущенные Pods могут продолжать работать, но:

- `kubectl apply` не работает;
- scheduler не работает;
- controllers не работают;
- новые Pods глобально не планируются;
- self-healing уровня кластера ограничен.

### HA Control Plane

Несколько control-plane Nodes создаются заранее:

```text
Control Plane
├── cp-1
├── cp-2
└── cp-3
```

Перед API servers ставится Load Balancer или VIP. Workers используют один адрес:

```text
api.cluster.local:6443
```

Произвольный worker автоматически control plane не становится.

---

<a id="autoscaling"></a>

## Масштабирование

### HPA

Horizontal Pod Autoscaler меняет количество Pod по метрикам.

### VPA

Vertical Pod Autoscaler изменяет или рекомендует requests и limits.

### Cluster Autoscaler

Изменяет количество Nodes.

```text
HPA
→ масштабирует Pods

Cluster Autoscaler
→ масштабирует Nodes
```

---

<a id="rollout"></a>

## Обновление приложения

Rolling update:

```text
v1 v1 v1
↓
v2 v1 v1
↓
v2 v2 v1
↓
v2 v2 v2
```

Deployment контролирует, сколько старых Pod можно удалить, сколько новых создать и проходят ли новые Pod readiness.

Если новая версия сломана, возможен rollback.

---

<a id="kubernetes-vs-compose"></a>

## Kubernetes и Docker Compose

| Docker Compose                       | Kubernetes                       |
| ------------------------------------ | -------------------------------- |
| Обычно одна машина                   | Одна или много Nodes             |
| Основная единица — container/service | Основная единица — Pod           |
| `docker compose up`                  | `kubectl apply`                  |
| Docker network                       | CNI + Pod network + Service      |
| Docker volume                        | Volume + PV + PVC + StorageClass |
| Ограниченный self-healing            | Постоянный reconciliation        |
| Нет scheduler между серверами        | Scheduler выбирает Node          |
| Нет полноценного rollout             | Deployment rolling update        |

Compose удобнее для локальной разработки. Kubernetes нужен при требованиях к нескольким Nodes, отказоустойчивости, масштабированию и управлению большим числом сервисов.

---

<a id="what-kubernetes-does-not-do"></a>

## Что Kubernetes не делает

Kubernetes обычно не:

- собирает image;
- хранит image;
- пишет Dockerfile;
- исправляет ошибки приложения;
- автоматически делает PostgreSQL отказоустойчивым;
- автоматически делает backup;
- заменяет CI/CD;
- заменяет мониторинг;
- заменяет бизнес-архитектуру;
- делает операции идемпотентными.

Обычная цепочка:

```text
Git push
   ↓
CI запускает тесты
   ↓
CI собирает image
   ↓
CI отправляет image в registry
   ↓
CD обновляет Deployment
   ↓
Kubernetes выполняет rollout
```

---

<a id="full-picture"></a>

## Полная картина работы Kubernetes

### Инфраструктура

```text
Server 1 → Control Plane Node
Server 2 → Worker Node
Server 3 → Worker Node
Server 4 → Worker Node
```

### Компоненты

```text
Control Plane
├── kube-apiserver
├── etcd
├── scheduler
└── controller-manager

Workers
├── kubelet
├── containerd
├── CNI
├── kube-proxy или замена
└── Pods
```

### Развёртывание

```text
Developer
   ↓
kubectl / CI/CD
   ↓
kube-apiserver
   ↓
etcd
   ↓
controllers
   ↓
scheduler
   ↓
kubelet
   ↓
container runtime
   ↓
Pod
   ↓
Container
   ↓
Process
```

### Сетевой запрос

```text
Internet
   ↓
Load Balancer
   ↓
Ingress Controller
   ↓
Service
   ↓
Ready Pod
   ↓
Container
   ↓
Application Process
```

### Внутренний запрос

```text
report-service Pod
        ↓
postgres-rw Service
        ↓
current PostgreSQL primary
```

### Storage

```text
Pod
 ↓
PVC
 ↓
PV
 ↓
real storage
```

### Управление состоянием

```text
spec
→ что хотим

status
→ что есть

controller
→ устраняет разницу
```

---

<a id="interview-questions"></a>

## Частые вопросы на собеседовании

### Что такое Kubernetes?

Система оркестрации контейнеризированных приложений, которая управляет размещением, масштабированием, сетью, обновлениями и восстановлением workload в кластере машин.

### Что такое Node?

Физическая или виртуальная машина, включённая в кластер Kubernetes.

### Что такое Pod?

Минимальная единица развёртывания Kubernetes, содержащая один или несколько контейнеров с общей сетью и жизненным циклом.

### Чем Pod отличается от контейнера?

Container запускает процесс. Pod является Kubernetes-объектом и содержит один или несколько контейнеров.

### Что делает Control Plane?

Хранит состояние кластера, принимает API-запросы, запускает controllers и принимает решения о размещении Pod.

### Что делает kube-apiserver?

Является центральной точкой Kubernetes API.

### Что делает etcd?

Хранит состояние Kubernetes API.

### Что делает scheduler?

Выбирает Node для нового Pod.

### Что делает kubelet?

На каждой Node обеспечивает запуск и работу назначенных ей Pod.

### Что делает container runtime?

Скачивает image и запускает контейнеры.

### Что такое Deployment?

Controller для stateless-приложений, управляющий количеством Pod, rollout и rollback.

### Что такое ReplicaSet?

Поддерживает заданное число одинаковых Pod. Обычно управляется Deployment.

### Что такое Service?

Стабильный сетевой endpoint для динамической группы Pod.

### Что такое Ingress?

Объект с HTTP/HTTPS-правилами маршрутизации к Services.

### Что такое Ingress Controller?

Программа, которая читает Ingress и реально проксирует трафик.

### Что такое CNI, CSI и CRI?

```text
CNI → сеть
CSI → storage
CRI → container runtime
```

### Что такое PV и PVC?

PV представляет постоянное хранилище, PVC является запросом приложения на хранилище.

### Request и limit — в чём разница?

Request используется scheduler для размещения. Limit задаёт верхнюю границу использования ресурса.

### Readiness и liveness — в чём разница?

Readiness определяет готовность принимать трафик. Liveness определяет, нужно ли перезапустить контейнер.

### Что будет, если Pod упал?

Если Pod управляется Deployment/ReplicaSet, контроллер создаст новый.

### Что будет, если Node упала?

Control Plane может создать заменяющие Pods на других Nodes.

### Что будет, если единственный Control Plane упал?

Workers автоматически не станут control plane. Работающие Pods могут продолжить работу, но глобальное управление кластером будет недоступно.

### Как сделать Control Plane отказоустойчивым?

Заранее создать несколько control-plane Nodes, использовать HA etcd и стабильный API endpoint через Load Balancer или VIP.

### StatefulSet делает failover PostgreSQL?

Нет. StatefulSet даёт стабильную идентичность и storage, но для election и failover нужен PostgreSQL cluster manager или Operator.

### Как приложение узнаёт новый primary базы?

Приложение подключается к стабильному Service, например `postgres-rw`. Operator обновляет Service или labels после смены primary. Приложение переподключается по тому же DNS-имени.

### Что такое Operator?

Специализированный Controller, который понимает предметную логику конкретной системы: PostgreSQL, Kafka, Redis и так далее.

---

<a id="cheat-sheet"></a>

## Краткая шпаргалка

```text
Kubernetes
= оркестратор контейнеризированных приложений
```

```text
Cluster
= Control Plane + Worker Nodes
```

```text
Node
= физический сервер или VM
```

```text
Pod
= один или несколько контейнеров
  с общей сетью и жизненным циклом
```

```text
Control Plane
= принимает решения и хранит состояние
```

```text
Worker Node
= запускает Pods
```

```text
API server
= центральная точка Kubernetes API
```

```text
etcd
= база состояния Kubernetes
```

```text
scheduler
= выбирает Node для Pod
```

```text
kubelet
= обеспечивает запуск Pod на Node
```

```text
container runtime
= запускает контейнеры
```

```text
Deployment
= управляет stateless Pods и rollout
```

```text
ReplicaSet
= поддерживает количество Pod
```

```text
StatefulSet
= стабильная идентичность и storage
  для stateful workload
```

```text
Service
= стабильный адрес группы Pod
```

```text
Ingress
= правила HTTP/HTTPS-маршрутизации
```

```text
Ingress Controller
= реальный HTTP reverse proxy
```

```text
CNI = сеть
CSI = storage
CRI = runtime
```

```text
ConfigMap = обычная конфигурация
Secret = чувствительная конфигурация
```

```text
Request
= сколько ресурсов учитывать при scheduling

Limit
= верхний предел использования
```

```text
Readiness
= можно ли направлять трафик

Liveness
= нужно ли перезапустить контейнер
```

```text
PV
= представленное Kubernetes хранилище

PVC
= запрос приложения на хранилище
```

```text
Operator
= controller с логикой конкретного приложения
```

```text
spec
= желаемое состояние

status
= фактическое состояние

reconciliation
= постоянное устранение разницы
  между spec и status
```

```text
Главная цепочка:

kubectl
→ API server
→ etcd
→ controllers
→ scheduler
→ kubelet
→ runtime
→ Pod
→ Container
→ Process
```
