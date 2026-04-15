.PHONY: test generate coverage install-linter lint lint-fix install-mockgen mocks mocks-contacts mocks-profile

MOCKGEN := $(shell go env GOPATH)/bin/mockgen
GOLANGCI_LINT := $(shell go env GOPATH)/bin/golangci-lint

test: generate mocks
	go test ./...

generate: $(MOCKGEN)
	PATH="$(shell go env GOPATH)/bin:$${PATH}" go generate ./...

coverage: generate
	go test $(COVER_PKGS) -coverprofile=coverage.out
	go tool cover -func=coverage.out

install-linter:
	@echo "Устанавливаем golangci-lint..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

lint: $(GOLANGCI_LINT)
	@echo "Запускаем линтеры..."
	PATH="$(shell go env GOPATH)/bin:$${PATH}" golangci-lint run ./...

lint-fix: $(GOLANGCI_LINT)
	PATH="$(shell go env GOPATH)/bin:$${PATH}" golangci-lint run ./... --fix

COVER_PKGS := $(shell go list ./... | grep -v '/mock$$')

install-mockgen:
	go install github.com/golang/mock/mockgen@v1.6.0

$(MOCKGEN):
	@$(MAKE) install-mockgen

$(GOLANGCI_LINT):
	@$(MAKE) install-linter
