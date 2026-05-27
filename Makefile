.PHONY: test generate coverage install-linter lint lint-fix install-mockgen install-easyjson mocks mocks-contacts mocks-profile proto install-proto-tools

MOCKGEN := $(shell go env GOPATH)/bin/mockgen
EASYJSON := $(shell go env GOPATH)/bin/easyjson
GOLANGCI_LINT := $(shell go env GOPATH)/bin/golangci-lint
PROTOC_GEN_GO := $(shell go env GOPATH)/bin/protoc-gen-go
PROTOC_GEN_GO_GRPC := $(shell go env GOPATH)/bin/protoc-gen-go-grpc
PROTO_DIR := api/proto
GEN_DIR := gen/go
PROTO_FILES := $(shell rg --files $(PROTO_DIR) -g '*.proto')

test: generate
	go test ./...

generate: $(MOCKGEN) $(EASYJSON)
	PATH="$(shell go env GOPATH)/bin:$${PATH}" go generate ./...

coverage: generate
	go test $(COVER_PKGS) -coverprofile=coverage.raw.out
	grep -Ev '(^|/)(mock|gen|dto|docs)(/|$$)|(^|/)dto\.go:|_easyjson\.go:' coverage.raw.out > coverage.out
	rm -f coverage.raw.out
	go tool cover -func=coverage.out

install-linter:
	@echo "Устанавливаем golangci-lint..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

lint: generate $(GOLANGCI_LINT)
	@echo "Запускаем линтеры..."
	PATH="$(shell go env GOPATH)/bin:$${PATH}" golangci-lint run ./...

lint-fix: $(GOLANGCI_LINT)
	PATH="$(shell go env GOPATH)/bin:$${PATH}" golangci-lint run ./... --fix

COVER_PKGS := $(shell go list ./... | grep -Ev '/mock$$|/gen(/|$$)|/dto(/|$$)|/cmd(/|$$)|/docs$$|/tools(/|$$)|/transport/grpc/clients/|/transport/subscription$$|/transport/ws$$|/gateway/ws$$')

install-mockgen:
	go install github.com/golang/mock/mockgen@v1.6.0

install-easyjson:
	go install github.com/mailru/easyjson/easyjson@v0.9.2

$(MOCKGEN):
	@$(MAKE) install-mockgen

$(EASYJSON):
	@$(MAKE) install-easyjson

$(GOLANGCI_LINT):
	@$(MAKE) install-linter

install-proto-tools:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

$(PROTOC_GEN_GO) $(PROTOC_GEN_GO_GRPC):
	@$(MAKE) install-proto-tools

proto: $(PROTOC_GEN_GO) $(PROTOC_GEN_GO_GRPC)
	PATH="$(shell go env GOPATH)/bin:$${PATH}" protoc -I $(PROTO_DIR) \
		--go_out=. --go_opt=module=github.com/go-park-mail-ru/2026_1_ASAP --go_opt=paths=import \
		--go-grpc_out=. --go-grpc_opt=module=github.com/go-park-mail-ru/2026_1_ASAP --go-grpc_opt=paths=import \
		$(PROTO_FILES)
