-- +goose Up
CREATE TABLE IF NOT EXISTS trips (
                                     id UUID PRIMARY KEY,
                                     user_id UUID NOT NULL,
                                     driver_id UUID NOT NULL,
                                     start_latitude DOUBLE PRECISION NOT NULL,
                                     start_longitude DOUBLE PRECISION NOT NULL,
                                     end_latitude DOUBLE PRECISION NOT NULL,
                                     end_longitude DOUBLE PRECISION NOT NULL,
                                     price BIGINT NOT NULL CHECK (price >= 0),
    status VARCHAR(32) NOT NULL,
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ,
    last_position_at TIMESTAMPTZ
    );

CREATE UNIQUE INDEX IF NOT EXISTS idx_trips_driver_active
    ON trips (driver_id)
    WHERE status = 'active';

-- +goose Down
DROP INDEX IF EXISTS idx_trips_driver_active;
DROP TABLE IF EXISTS trips;