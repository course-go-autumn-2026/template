.PHONY: generate lint test test-integration migrate build run

generate:
	go tool oapi-codegen \
		-generate types,chi-server \
		-package api \
		-o internal/generated/api.gen.go \
		contracts/openapi/trip-service.openapi.yaml
	go generate ./...

lint:
	golangci-lint run ./...

test:
	go test -race -coverprofile=coverage.out -covermode=atomic ./internal/domain/... ./internal/usecase/... ./internal/http/...
	go tool cover -func=coverage.out

test-integration:
	go test -v -tags=integration ./internal/repository/...

migrate:
	goose -dir migrations postgres "$$DATABASE_URL" up

build:
	go build -o bin/service cmd/service/main.go

run: build
	./bin/service
