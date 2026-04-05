.PHONY: test coverage install-linter lint lint-fix install-mockgen mocks mocks-profile

test:
	go test ./...

coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out

# Получить бинарник линтера (установит в проект, не глобально)
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

install-mockgen:
	go install github.com/golang/mock/mockgen@v1.6.0

$(MOCKGEN):
	@$(MAKE) install-mockgen


mocks: mocks-contacts mocks-profile


mocks-profile: internal/services/profile/mock/profile_mock.go
mocks-contacts: internal/services/contacts/mock/contacts_mock.go

internal/services/profile/mock/profile_mock.go: internal/services/profile/profile.go | $(MOCKGEN)
	@echo "Generating mocks for profile..."
	@mkdir -p $(dir $@)
	$(MOCKGEN) -source=$< -destination=$@ -package=mock_profile

internal/services/contacts/mock/contacts_mock.go: internal/services/contacts/contacts.go | $(MOCKGEN)
	@echo "Generating mocks for contacts..."
	@mkdir -p $(dir $@)
	$(MOCKGEN) -source=$< -destination=$@ -package=mock_contacts