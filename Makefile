SHELL := /bin/bash

BINARY_NAME := trip-service
OUT_DIR := out
OPENAPI_SPEC := contracts/openapi/trip-service.openapi.yaml
OPENAPI_OUTPUT := internal/generated/api.gen.go

.PHONY: build run test clean deps migrate migrate-down generate check

build:
	mkdir -p $(OUT_DIR)
	go build -o $(OUT_DIR)/$(BINARY_NAME) ./cmd/trip-service

run:
	set -a; source .env.example; source .env; set +a; go run ./cmd/trip-service

test:
	go test -race ./...

migrate:
	set -a; source .env.example; source .env; set +a; go run ./cmd/migrator up

migrate-down:
	set -a; source .env.example; source .env; set +a; go run ./cmd/migrator down

generate:
	mkdir -p $(dir $(OPENAPI_OUTPUT))
	go tool oapi-codegen \
		-generate types,gin-server \
		-include-operation-ids createTrip,getTrip,finishTrip,health,ready \
		-package api \
		-o $(OPENAPI_OUTPUT) \
		$(OPENAPI_SPEC)

deps:
	go mod tidy

check: build test

clean:
	rm -rf $(OUT_DIR)
