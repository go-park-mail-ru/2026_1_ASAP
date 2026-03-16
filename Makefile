.PHONY: test coverage install-linter lint lint-fix

test:
	go test ./...

coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out

# Получить бинарник линтера (установит в проект, не глобально)
.PHONY: install-linter
install-linter:
	@echo "Устанавливаем golangci-lint..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Запустить линтер
.PHONY: lint
lint:
	@echo "Запускаем линтеры..."
	golangci-lint run ./...

# Автоисправление (где возможно)
.PHONY: lint-fix
lint-fix:
	golangci-lint run ./... --fix
