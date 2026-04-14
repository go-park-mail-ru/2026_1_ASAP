.PHONY: test generate coverage install-linter lint lint-fix install-mockgen mocks mocks-contacts mocks-profile

test: generate mocks
	go test ./...

generate: $(MOCKGEN)
	PATH="$(shell go env GOPATH)/bin:$${PATH}" go generate ./...
	@$(MAKE) mocks-contacts

coverage: generate
	go test $(COVER_PKGS) -coverprofile=coverage.out
	go tool cover -func=coverage.out

install-linter:
	@echo "Устанавливаем golangci-lint..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Запустить линтер
lint:
	@echo "Запускаем линтеры..."
	golangci-lint run ./...

# Автоисправление (где возможно)
lint-fix:
	golangci-lint run ./... --fix

MOCKGEN := $(shell go env GOPATH)/bin/mockgen

COVER_PKGS := $(shell go list ./... | grep -v '/mock$$')

install-mockgen:
	go install github.com/golang/mock/mockgen@v1.6.0

$(MOCKGEN):
	@$(MAKE) install-mockgen


mocks: mocks-contacts


mocks-contacts: internal/services/contacts/mock/contacts_mock.go


internal/services/contacts/mock/contacts_mock.go: internal/services/contacts/contacts.go | $(MOCKGEN)
	@echo "Generating mocks for contacts..."
	@mkdir -p $(dir $@)
	$(MOCKGEN) -source=$< -destination=$@ -package=mock_contacts