-- +goose Up
CREATE TABLE trips (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    driver_id UUID NOT NULL,
    start_lat DOUBLE PRECISION NOT NULL,
    start_lon DOUBLE PRECISION NOT NULL,
    end_lat DOUBLE PRECISION NOT NULL,
    end_lon DOUBLE PRECISION NOT NULL,
    price BIGINT NOT NULL,
    status TEXT NOT NULL,
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ,
    last_position_at TIMESTAMPTZ
);

CREATE TABLE trip_status_history (
    id SERIAL PRIMARY KEY,
    trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    status TEXT NOT NULL,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_active_driver ON trips (driver_id) WHERE status = 'active';

-- +goose Down