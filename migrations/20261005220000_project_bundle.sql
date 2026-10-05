-- +goose Up
ALTER TABLE projects ADD COLUMN bundle_prefix TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE projects DROP COLUMN IF EXISTS bundle_prefix;
