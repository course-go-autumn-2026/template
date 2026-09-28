-- +goose Up
-- +goose StatementBegin
-- 1. Таблица поездок
CREATE TABLE trips (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    driver_id UUID NOT NULL,
    price BIGINT NOT NULL,
    status VARCHAR(50) NOT NULL,
    started_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    finished_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_trips_driver_active_unique 
ON trips(driver_id) 
WHERE status = 'active';

-- 2. Таблица истории статусов
CREATE TABLE trip_status_history (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    trip_id UUID NOT NULL,
    status VARCHAR(50) NOT NULL,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_status_history_trip FOREIGN KEY (trip_id) REFERENCES trips(id) ON DELETE CASCADE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS trip_status_history;
DROP TABLE IF EXISTS trips;
-- +goose StatementEnd

