-- +goose Up
ALTER TABLE trips
    ADD COLUMN IF NOT EXISTS last_position_at TIMESTAMPTZ;

CREATE TABLE IF NOT EXISTS trip_positions (
    id          BIGSERIAL PRIMARY KEY,
    trip_id     UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    latitude    DOUBLE PRECISION NOT NULL,
    longitude   DOUBLE PRECISION NOT NULL,
    recorded_at TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_trip_positions_trip_recorded
    ON trip_positions (trip_id, recorded_at ASC, id ASC);

CREATE INDEX IF NOT EXISTS idx_trips_active_stale
    ON trips (COALESCE(last_position_at, started_at) ASC)
    WHERE status = 'active';

-- +goose Down
DROP INDEX IF EXISTS idx_trips_active_stale;
DROP INDEX IF EXISTS idx_trip_positions_trip_recorded;
DROP TABLE IF EXISTS trip_positions;
