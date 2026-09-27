#start
run:
	set -a; . ./.env; set +a; go run ./cmd/trip-service
#generating code from openapi spec	
generate:
	go get -tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen

	go tool oapi-codegen \
	-generate types,chi-server \
	-package api \
	-o internal/generated/api.gen.go \
	contracts/openapi/trip-service.openapi.yaml