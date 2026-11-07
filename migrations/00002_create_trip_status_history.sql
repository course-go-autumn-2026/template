-- +goose Up
CREATE TABLE IF NOT EXISTS trip_status_history (
                                                   id BIGSERIAL PRIMARY KEY,
                                                   trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    status VARCHAR(32) NOT NULL,
    changed_at TIMESTAMPTZ NOT NULL
    );

CREATE INDEX IF NOT EXISTS idx_trip_status_history_trip_id
    ON trip_status_history (trip_id);

-- +goose Down
DROP TABLE IF EXISTS trip_status_history;