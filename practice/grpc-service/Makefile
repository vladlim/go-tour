GO ?= go
PROTOC ?= protoc
GO_BIN := $(shell $(GO) env GOPATH)/bin

.PHONY: generate install-tools

# protoc must be installed separately; plugin versions are pinned.
install-tools:
	$(GO) install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11
	$(GO) install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.6.0

generate:
	PATH="$(GO_BIN):$$PATH" $(PROTOC) \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		api/user/v1/user.proto
