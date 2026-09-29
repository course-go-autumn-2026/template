.PHONY: generate migrate-up migrate-down run build lint test

BIN_DIR := $(CURDIR)/bin

generate:
	@mkdir -p internal/generated
	go tool oapi-codegen \
		-generate types,chi-server \
		-package api \
		-o internal/generated/api.gen.go \
		contracts/openapi/trip-service.openapi.yaml

migrate-up:
	@test -n "$$DATABASE_URL" || (echo "DATABASE_URL is not set" && exit 1)
	go tool goose -dir migrations postgres "$$DATABASE_URL" up

migrate-down:
	@test -n "$$DATABASE_URL" || (echo "DATABASE_URL is not set" && exit 1)
	go tool goose -dir migrations postgres "$$DATABASE_URL" down

build:
	go build -o $(BIN_DIR)/trip-service ./cmd/trip-service

run:
	go run ./cmd/trip-service/main.go