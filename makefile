.PHONY: generate migrate run

generate:
	go tool oapi-codegen -generate types,chi-server -package api -o internal/generated/api.gen.go contracts/openapi/trip-service.openapi.yaml

migrate:
	psql "$$(grep DATABASE_URL .env | cut -d '=' -f2-)" -f migrations/00001_init.sql

run:
	go run ./cmd/app