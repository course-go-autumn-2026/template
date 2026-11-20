-- +goose Up
CREATE TABLE idempotency_keys (
    idempotency_key UUID PRIMARY KEY,
    request_hash    BYTEA NOT NULL,
    response_body   BYTEA NOT NULL,
    content_type    TEXT NOT NULL,
    location        TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at      TIMESTAMPTZ NOT NULL,

    CHECK (expires_at > created_at)
);

CREATE INDEX idempotency_keys_expires_at_idx
    ON idempotency_keys (expires_at);

-- +goose Down
DROP TABLE IF EXISTS idempotency_keys;
