# Простой gRPC-сервис

Требуется Go 1.22 или новее. Сервис создаёт пользователей и находит их по ID.
Данные хранятся в памяти и исчезают при перезапуске. Соединение без TLS — для локального обучения.

## Три слоя

```text
gRPC-клиент
    ↓
internal/transport/grpc  — принимает protobuf-запросы, преобразует ошибки в gRPC-коды
    ↓
internal/service         — бизнес-логика: проверяет и нормализует имя и ID
    ↓
internal/repository      — хранит пользователей в map под mutex
```

`internal/user` содержит общую модель и ошибки. Слои связаны через интерфейсы,
поэтому хранилище можно заменить, например, на PostgreSQL. Зависимости собираются
в `cmd/server/main.go`. Бизнес-логика не зависит от gRPC и protobuf.

## Запуск

Из этой директории:

```sh
go mod download
go run ./cmd/server
```

В другом терминале из той же директории:

```sh
go run ./cmd/client -name "Vlad"
```

Ожидаемый результат первого вызова:

```text
Created: id=1 name=Vlad
Found:   id=1 name=Vlad
```

Сервер по умолчанию слушает `127.0.0.1:50051`. Флаг `-addr` есть у сервера и клиента.
Ctrl+C завершает сервер с ожиданием активных запросов до 5 секунд.

## Вызовы через grpcurl (необязательно)

Включён server reflection, поэтому передавать proto-файл не требуется:

```sh
grpcurl -plaintext -d '{"name":"Vlad"}' localhost:50051 user.v1.UserService/CreateUser
grpcurl -plaintext -d '{"id":"1"}' localhost:50051 user.v1.UserService/GetUser
```

Пустое имя или ID возвращают `InvalidArgument`, неизвестный ID — `NotFound`.

## Проверка и генерация

```sh
go test -race ./...
go vet ./...
```

Сгенерированные `*.pb.go` уже включены, для запуска protoc не нужен.
Для генерации нужен установленный `protoc`. Один раз установите Go-плагины:

```sh
make install-tools
```

После изменения `api/user/v1/user.proto`:

```sh
make generate
```

Цель `generate` сама добавляет `$(go env GOPATH)/bin` в PATH для поиска плагинов.
