CREATE TABLE idempotency_keys (
  key UUID PRIMARY KEY,
  request_hash BYTEA NOT NULL CHECK (octet_length(request_hash) = 32),
  trip_id UUID REFERENCES trips(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idempotency_keys_expires_at_idx ON idempotency_keys (expires_at);