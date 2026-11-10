-- +goose Up
CREATE TABLE trip_idempotency (
    idempotency_key UUID PRIMARY KEY,
    request_hash BYTEA NOT NULL CHECK (octet_length(request_hash) = 32),
    trip_id UUID REFERENCES trips(id) ON DELETE CASCADE,
    response JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    CHECK ((trip_id IS NULL) = (response IS NULL))
);

CREATE INDEX trip_idempotency_expires_at_idx ON trip_idempotency (expires_at);

-- +goose Down
DROP TABLE trip_idempotency;
