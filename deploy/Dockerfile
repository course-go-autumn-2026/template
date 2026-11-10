FROM golang:1.24.13-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY api ./api
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/trip-service ./cmd/trip-service

FROM scratch
COPY --from=build /out/trip-service /trip-service
USER 65532:65532
ENTRYPOINT ["/trip-service"]
