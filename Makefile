ifneq (,$(wildcard ./.env))
include .env
export
endif

.PHONY: migrate migrate-up migrate-down migrate-status generate run build test test-integration lint

migrate: migrate-up

migrate-up:
	go tool goose -dir ./migrations postgres "$$DATABASE_URL" up

migrate-down:
	go tool goose -dir ./migrations postgres "$$DATABASE_URL" down-to 0

migrate-status:
	go tool goose -dir ./migrations postgres "$$DATABASE_URL" status

generate:
	go tool oapi-codegen \
		-generate types,chi-server \
		-package api \
		-o internal/generated/api.gen.go \
		contracts/openapi/trip-service.openapi.yaml

run:
	go run ./cmd/trip-service

build:
	go build ./...

lint:
	golangci-lint run

test:
	go test -coverprofile=coverage.out ./internal/domain/... ./internal/usecase/...

test-integration:
	go test -tags=integration ./...
