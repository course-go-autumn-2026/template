GO ?= go

include .env.example
-include .env
export

.PHONY: generate build run test test-integration lint migrate migrate-down migrate-status

generate:
	$(GO) tool oapi-codegen -config api/oapi-codegen.yaml contracts/openapi/trip-service.openapi.yaml
	$(GO) generate ./...

build:
	$(GO) build -o bin/trip-service ./cmd/trip-service

run:
	$(GO) run ./cmd/trip-service

test:
	$(GO) test -race ./...

test-integration:
	$(GO) test -tags integration -race ./...

lint:
	golangci-lint run

migrate:
	$(GO) tool goose -dir migrations postgres "$$DATABASE_URL" up

migrate-down:
	$(GO) tool goose -dir migrations postgres "$$DATABASE_URL" down

migrate-status:
	$(GO) tool goose -dir migrations postgres "$$DATABASE_URL" status
