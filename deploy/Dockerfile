# syntax=docker/dockerfile:1.7

FROM golang:1.26-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY cmd ./cmd
COPY internal ./internal

RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build \
    -trimpath -buildvcs=false -ldflags="-s -w" \
    -o /out/trip-service ./cmd/trip-service

FROM build AS goose-build
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build \
    -trimpath -buildvcs=false -ldflags="-s -w" \
    -o /out/goose github.com/pressly/goose/v3/cmd/goose

FROM gcr.io/distroless/static-debian12:nonroot AS migrator
COPY --from=goose-build --chown=65532:65532 /out/goose /goose
COPY --chown=65532:65532 migrations /migrations
USER 65532:65532
ENTRYPOINT ["/goose"]

FROM gcr.io/distroless/static-debian12:nonroot AS runtime
COPY --from=build --chown=65532:65532 /out/trip-service /trip-service
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/trip-service"]
