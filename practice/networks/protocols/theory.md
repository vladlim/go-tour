# Конспект: сети и протоколы

## Общая картина

Передача обычно выглядит так:

```text
приложение:  DNS, HTTP, WebSocket, gRPC
                 ↓
безопасность:       TLS
                 ↓
транспорт:       TCP или UDP
                 ↓
сеть:                IP
                 ↓
канал:       Ethernet / Wi‑Fi
```

Инкапсуляция означает, что HTTP-сообщение становится данными TLS record, тот —
данными TCP segment, тот — payload IP packet, а пакет помещается в Ethernet frame.
На принимающей стороне слои снимаются в обратном порядке. Модель условна: TLS
расположен между приложением и транспортом, а HTTP/3 работает поверх QUIC/UDP.

Полезно добавить к исходному списку: Ethernet, ARP/NDP, ICMP, DHCP, NAT, routing,
порты и сокеты, subnet/CIDR; затем HTTP/2, HTTP/3 и QUIC. Для backend-разработчика
ещё важны proxy/reverse proxy, load balancing, timeout/retry/backoff, idempotency,
connection pooling и диагностика через `dig`, `curl`, `ss`, `traceroute`, `tcpdump`.

## IP

IP доставляет пакеты между адресами без гарантии доставки, порядка или отсутствия
дубликатов. IPv4 имеет 32-битный адрес, IPv6 — 128-битный. CIDR `192.168.1.10/24`
означает 24 бита сети; хост находится в сети `192.168.1.0/24`. Router выбирает
следующий hop по таблице маршрутов. TTL/Hop Limit не даёт пакету вечно ходить по
циклу; ICMP сообщает диагностические ошибки. NAT преобразует адреса/порты на
границе сети, но сам не является firewall.

Приложение обычно работает не с IP-пакетом, а с socket: сочетанием transport,
локального IP и порта. Соединение TCP идентифицирует 4-tuple: source IP/port и
destination IP/port. См. `go run ./cmd/ip`.

## TCP

TCP — надёжный упорядоченный двунаправленный поток байтов. Handshake: `SYN →
SYN-ACK → ACK`. Sequence number, acknowledgement и retransmission обеспечивают
порядок и повторную доставку; flow control защищает получателя, congestion control
— сеть. Закрытие каждого направления независимо использует FIN/ACK; RST аварийно
обрывает соединение.

Главная ловушка: TCP не сохраняет границы сообщений. Один `Write` не равен одному
`Read`, поэтому протокол задаёт framing: delimiter, фиксированный размер или
length-prefix. Всегда нужны deadlines/timeouts; retry после неоднозначного сбоя
безопасен только для идемпотентной операции. `cmd/tcp` использует newline framing
и `CloseWrite`, чтобы показать half-close и EOF.

## UDP

UDP отправляет отдельные датаграммы. Handshake, подтверждения, порядка и повторов
нет; зато меньше состояния и head-of-line blocking между датаграммами. Датаграмма
может потеряться, продублироваться или прийти не по порядку. Если buffer мал, хвост
датаграммы отбрасывается. Больших сообщений лучше избегать из-за IP fragmentation.

UDP используют DNS, media, games и QUIC. Надёжность при необходимости строит
прикладной протокол (ID, ACK, retry, дедупликация). См. `cmd/udp`.

## DNS

DNS — распределённая иерархическая база имён. Stub resolver спрашивает recursive
resolver, а тот при отсутствии cache проходит root → TLD → authoritative server.
Записи: `A` (IPv4), `AAAA` (IPv6), `CNAME` (alias), `MX`, `TXT`, `NS`, `SRV`, `PTR`.
TTL определяет время кеширования; отрицательные ответы тоже кешируются.

Обычный DNS использует UDP/53, при truncation и zone transfer — TCP; DoT шифрует
его TLS, DoH переносит через HTTPS. DNS сообщает адрес, но не проверяет доступность
сервиса. Go `net.Resolver` учитывает настройки ОС; результат может различаться при
pure-Go и cgo resolver. См. `cmd/dns`.

## TLS и HTTPS

TLS даёт конфиденциальность, целостность и аутентификацию peer. Упрощённый TLS 1.3
handshake: ClientHello (versions, cipher suites, SNI, ALPN, key share), ServerHello,
сертификат и доказательство владения ключом, затем стороны выводят session keys.
CA-подпись и hostname verification образуют цепочку доверия. После handshake
симметричное шифрование быстрее асимметричного.

TLS не скрывает IP и обычно SNI; он не спасает от скомпрометированного endpoint.
Нельзя использовать `InsecureSkipVerify` как production-решение. HTTPS — HTTP
внутри TLS. ALPN согласует, например, `h2` или `http/1.1`. См. `cmd/tls` и `cmd/http`.

## HTTP

HTTP — stateless request/response протокол. Метод и target, headers и необязательное
body образуют request; status code, headers и body — response. `GET/HEAD/OPTIONS`
считаются safe; `PUT` и `DELETE` идемпотентны по семантике, `POST` обычно нет.
Классы кодов: 1xx information, 2xx success, 3xx redirect, 4xx client error, 5xx
server error. Cache-Control/ETag управляют cache, Content-Type описывает формат.

HTTP/1.1 текстовый и переиспользует TCP; pipelining практически не используется.
HTTP/2 мультиплексирует бинарные streams в одном TCP и сжимает headers (HPACK),
но потеря TCP packet задерживает все streams. HTTP/3 переносит HTTP через QUIC
поверх UDP, устраняя этот межпоточный transport HOL. Server должен ограничивать
размеры, timeouts и корректно завершаться. См. `cmd/http`.

## WebSocket

Клиент начинает HTTP/1.1 Upgrade с случайным `Sec-WebSocket-Key`. Сервер возвращает
101 и `Base64(SHA1(key + GUID))` в `Sec-WebSocket-Accept`; далее соединение становится
дуплексным потоком frames. Client frames обязательно masked, server frames — нет.
Есть text/binary, continuation и control frames: ping, pong, close.

WebSocket не задаёт бизнес-формат, delivery guarantees или восстановление. После
обрыва клиент создаёт новое соединение с exponential backoff+jitter. Если нельзя
терять события, приложение передаёт sequence/resume token, сервер хранит историю,
а клиент дедуплицирует. Heartbeat ping/pong выявляет полумёртвые соединения. Backpressure
нужен, чтобы медленный клиент не съел память. См. `cmd/websocket`.

## gRPC

gRPC — RPC framework: контракт `.proto` задаёт services/messages, генератор создаёт
typed stubs, protobuf даёт бинарную сериализацию. Транспорт обычно HTTP/2: один
connection, много streams, binary framing, header compression. Виды RPC: unary,
server-streaming, client-streaming, bidirectional streaming. Metadata похожа на
headers; status состоит из gRPC code/message/details.

Deadline должен прийти от клиента и распространяться вниз через `context`; cancel
останавливает бесполезную работу. Retry разрешают только для подходящих операций.
Load balancing, health checking, keepalive и maximum message size требуют явной
настройки. В production используют protobuf; `cmd/grpc` намеренно регистрирует JSON
codec и ручной ServiceDesc, чтобы настоящий gRPC был запускаем без `protoc`, а
механика transport/metadata/status оставалась видимой.

## Как выбирать

HTTP/JSON удобен для публичного API и браузеров. gRPC хорош для типизированного
межсервисного API и streaming. WebSocket нужен для постоянного двунаправленного
канала с браузером. Raw TCP/UDP выбирают, когда вы проектируете собственный wire
protocol и готовы реализовать framing, совместимость, безопасность и эксплуатацию.

При диагностике двигайтесь снизу вверх: есть ли маршрут/IP → открыт ли порт и
состоялся ли TCP/UDP обмен → прошёл ли TLS → корректны ли HTTP/gRPC/WebSocket →
правильна ли бизнес-семантика.

## Практические задания после запуска

1. TCP: уберите `\n` у сообщения и объясните зависание до deadline; затем замените
   newline framing на 4-байтовый big-endian length-prefix.
2. UDP: поставьте server buffer в 3 байта и увидьте усечение; добавьте request ID,
   timeout и один retry, а на сервере — дедупликацию.
3. DNS: сравните `net.DefaultResolver` с `dig A example.com`, `dig AAAA` и `dig +trace`.
4. TLS: подключитесь с пустым root pool и изучите ошибку; посмотрите ClientHello и
   сертификат командой `openssl s_client -connect host:port -servername localhost`.
5. HTTP: добавьте POST, ограничьте body через `http.MaxBytesReader`, задайте server
   timeouts и сделайте graceful shutdown.
6. WebSocket: намеренно отправьте unmasked client frame; затем добавьте reconnect
   loop с backoff `1s, 2s, 4s` плюс jitter и sequence number в сообщениях.
7. gRPC: вызовите unary с уже истёкшим deadline, добавьте unary interceptor для
   request ID и сгенерируйте обычные protobuf stubs из `api/greeter.proto`.

Ответьте себе для каждого опыта: где границы сообщений, кто обнаруживает обрыв,
какие гарантии доставки есть, где выставлен timeout, можно ли безопасно повторить
операцию и что будет видно в логах/метриках.
