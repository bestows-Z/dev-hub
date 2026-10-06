-- +goose Up

ALTER TABLE projects
  ADD COLUMN runtime_bundle_key TEXT NOT NULL DEFAULT '',
  ADD COLUMN runtime_bundle_uploaded BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down

ALTER TABLE projects
  DROP COLUMN runtime_bundle_uploaded,
  DROP COLUMN runtime_bundle_key;
