FROM golang:1.25.12-alpine AS builder

WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/observability-demo \
    ./cmd/app

FROM scratch

COPY --from=builder /out/observability-demo /observability-demo

EXPOSE 8080

ENTRYPOINT ["/observability-demo"]
