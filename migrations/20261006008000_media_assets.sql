-- +goose Up
CREATE TABLE media_assets (
  id UUID PRIMARY KEY,
  owner_id BIGINT NOT NULL REFERENCES users(id),
  object_key TEXT NOT NULL UNIQUE,
  content_type VARCHAR(32) NOT NULL,
  size_bytes BIGINT NOT NULL CHECK (size_bytes > 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_media_assets_owner_created ON media_assets(owner_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS media_assets;
