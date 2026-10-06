-- +goose Up

ALTER TABLE users
  ADD COLUMN display_name VARCHAR(60) NOT NULL DEFAULT '',
  ADD COLUMN bio VARCHAR(500) NOT NULL DEFAULT '',
  ADD COLUMN website_url VARCHAR(255) NOT NULL DEFAULT '',
  ADD COLUMN avatar_uploaded BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down

ALTER TABLE users
  DROP COLUMN avatar_uploaded,
  DROP COLUMN website_url,
  DROP COLUMN bio,
  DROP COLUMN display_name;
