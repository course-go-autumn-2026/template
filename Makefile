ifneq (,$(wildcard .env))
include .env
export
endif

GO ?= go

.PHONY: help generate migrate migrate-down migrate-status run build test lint tidy

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

tidy:
	$(GO) mod tidy

generate:
	$(GO) tool oapi-codegen \
		-generate types,chi-server \
		-package api \
		-o internal/generated/api.gen.go \
		contracts/openapi/trip-service.openapi.yaml

migrate:
	$(GO) tool goose -dir migrations postgres "$(DATABASE_URL)" up

migrate-down:
	$(GO) tool goose -dir migrations postgres "$(DATABASE_URL)" down

migrate-status:
	$(GO) tool goose -dir migrations postgres "$(DATABASE_URL)" status

run:
	$(GO) run ./cmd/service

build:
	$(GO) build -o bin/trip-service ./cmd/service

test:
	$(GO) test -race -cover ./internal/...

test-integration:
	$(GO) test -race -tags=integration -count=1 ./internal/repository/...

lint:
	golangci-lint run
