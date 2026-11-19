-- +goose Up
ALTER TABLE users DROP COLUMN IF EXISTS athlete_code;

-- +goose Down
ALTER TABLE users ADD COLUMN athlete_code BIGINT NOT NULL DEFAULT 0;