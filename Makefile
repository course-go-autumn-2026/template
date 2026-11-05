.PHONY: run generate migrate migrate-down test

-include .env
export

run:
	go run ./cmd/trip-service

generate:
	go tool oapi-codegen -generate types,chi-server -package api -o internal/generated/api.gen.go contracts/openapi/trip-service.openapi.yaml

migrate:
	go tool goose -dir migrations postgres "$(DATABASE_URL)" up

migrate-down:
	go tool goose -dir migrations postgres "$(DATABASE_URL)" down

test:
	go test -race ./...