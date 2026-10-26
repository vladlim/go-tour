# Сети и протоколы — практикум 14.08

Материалы состоят из [конспекта](./theory.md) и маленьких демонстраций. Каждая
демонстрация сама поднимает сервер на случайном локальном порту, запускает клиент,
пишет этапы обмена в лог и завершается. Поэтому не нужно держать два терминала и
не возникает конфликтов портов.

## Быстрый старт

Нужен Go 1.22+. Из этой директории:

```bash
go mod download
go run ./cmd/ip
go run ./cmd/tcp
go run ./cmd/udp
go run ./cmd/dns
go run ./cmd/tls
go run ./cmd/http
go run ./cmd/websocket
go run ./cmd/grpc
```

Или `make run`. Для TCP/UDP можно параллельно открыть Wireshark на loopback-интерфейсе
с фильтром `tcp` или `udp`. TLS-демо использует временный самоподписанный сертификат,
а не отключает проверку сертификатов.

## Карта примеров

| Команда | Что наблюдать |
|---|---|
| `ip` | адрес, маску подсети, loopback и сетевые интерфейсы |
| `tcp` | listen → accept → stream read/write → half-close → EOF |
| `udp` | отдельные датаграммы, адрес отправителя, отсутствие соединения |
| `dns` | lookup через системный resolver, A/AAAA и reverse lookup |
| `tls` | TCP, TLS handshake, версия/cipher/SNI, проверку сертификата |
| `http` | request/response, заголовки, keep-alive, HTTPS поверх TLS |
| `websocket` | HTTP Upgrade, `Sec-WebSocket-Accept`, masking, ping/pong, close |
| `grpc` | unary и server-streaming RPC, metadata, status codes |

Все примеры учебные. Самописный WebSocket реализует только нужную для демонстрации
часть RFC 6455; в production следует использовать поддерживаемую библиотеку.
