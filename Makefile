ifneq (,$(wildcard .env))
    include .env
    export
endif

MIGRATIONS_DIR = ./migrations

.PHONY: create migrate migrate-up migrate-down run test generate

generate: 
	go tool oapi-codegen -generate types,chi-server -package api -o internal/generated/api.gen.go contracts/openapi/trip-service.openapi.yaml

create:
	goose -dir ${MIGRATIONS_DIR} create $(NAME) sql

migrate: migrate-up

migrate-up:
	goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" up
	
migrate-down:
	goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" down

run:
	go run ./cmd/main.go

test:
	go test -race ./...
