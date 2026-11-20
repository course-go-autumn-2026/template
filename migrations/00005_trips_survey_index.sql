-- +goose Up
CREATE INDEX IF NOT EXISTS trip_survey_index ON trips(
    (CASE WHEN last_position_at IS NOT NULL THEN last_position_at ELSE started_at END) ASC,
    id ASC
) WHERE status = 'active';

-- +goose Down
DROP INDEX IF EXISTS trip_survey_index;
