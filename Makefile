ifneq ($(wildcard .env),)
    include .env
    export
endif

.PHONY: generate migrate-up migrate-down run

# 1. Генерация кода по контракту OpenAPI
generate:
	go tool oapi-codegen \
	  -generate types,chi-server \
	  -package api \
	  -o internal/api/api.gen.go \
	  contracts/openapi/trip-service.openapi.yaml

# 2. Накатить миграции (используем DATABASE_URL из .env)
migrate:
	go tool goose -dir migrations postgres "$(DATABASE_URL)" up

# 3. Откатить миграции на один шаг назад
migrate-down:
	go tool goose -dir migrations postgres "$(DATABASE_URL)" down

# 4. Запуск сервиса локально
run:
	go run cmd/trip-service/main.go
