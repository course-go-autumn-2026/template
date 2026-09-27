FROM golang:1.26.1-alpine AS builder

WORKDIR /src

RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/trip-service \
    ./cmd/trip-service

RUN CGO_ENABLED=0 GOOS=linux \
    go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/migrator \
    ./cmd/migrator


FROM scratch

WORKDIR /app

COPY --from=builder /etc/ssl/certs/ca-certificates.crt \
    /etc/ssl/certs/ca-certificates.crt

COPY --from=builder --chown=65532:65532 \
    /out/trip-service \
    /app/trip-service

COPY --from=builder --chown=65532:65532 \
    /out/migrator \
    /app/migrator

USER 65532:65532

EXPOSE 8080

CMD ["/app/trip-service"]