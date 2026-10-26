.PHONY: run test fmt

DEMOS := ip tcp udp dns tls http websocket grpc

run:
	@for demo in $(DEMOS); do echo "\n===== $$demo ====="; go run ./cmd/$$demo || exit 1; done

test:
	go test ./...

fmt:
	gofmt -w ./cmd
