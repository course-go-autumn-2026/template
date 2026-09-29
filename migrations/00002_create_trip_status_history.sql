-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS trip_status_history (
                                                   id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    status VARCHAR(32) NOT NULL,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT now()
    );

CREATE INDEX idx_trip_status_history_trip_id ON trip_status_history (trip_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_trip_status_history_trip_id;
DROP TABLE IF EXISTS trip_status_history;
-- +goose StatementEnd