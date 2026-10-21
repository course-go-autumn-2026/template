SERVICE=trip-service

.PHONY: run migrate generate test test-integration lint build docker-build

run:
	set -a; \
	[ ! -f .env.example ] || . ./.env.example; \
	[ ! -f .env ] || . ./.env; \
	set +a; \
	go run ./cmd/trip-service

migrate:
	set -a; \
	[ ! -f .env ] || . ./.env; \
	set +a; \
	go tool goose -dir migrations postgres "$$DATABASE_URL" up

generate:
	go tool oapi-codegen -config config/oapi-codegen.yaml contracts/openapi/trip-service.openapi.yaml
	go generate ./internal/usecase/trip/... ./internal/http/...

test:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -n 1

test-integration:
	set -a; \
	[ ! -f .env ] || . ./.env; \
	set +a; \
	go test -race -tags=integration -count=2 ./internal/repository/...

lint:
	go tool golangci-lint run ./...

build:
	go build -o bin/$(SERVICE) ./cmd/trip-service

docker-build:
	docker build -f deploy/Dockerfile -t trip-service:lab2 .
