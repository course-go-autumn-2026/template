SHELL := /bin/bash

.PHONY: run migrate migrate-down generate build test test-integration lint

run:
	set -a; source .env.example; if [ -f .env ]; then source .env; fi; set +a; go run ./cmd/trip-service

migrate:
	set -a; source .env.example; if [ -f .env ]; then source .env; fi; set +a; go tool goose -dir migrations postgres "$$DATABASE_URL" up

migrate-down:
	set -a; source .env.example; if [ -f .env ]; then source .env; fi; set +a; go tool goose -dir migrations postgres "$$DATABASE_URL" down

generate:
	go tool oapi-codegen -config oapi-codegen.yaml contracts/openapi/trip-service.openapi.yaml
	go generate ./internal/...

build:
	go build ./...

test:
	go test -race -count=2 -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
test-integration:
	set -a; source .env.example; if [ -f .env ]; then source .env; fi; set +a; go test -race -count=2 -tags=integration ./internal/repository/...
lint:
	golangci-lint run
