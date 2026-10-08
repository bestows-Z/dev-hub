-- +goose Up
CREATE TABLE oauth_identities (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  provider VARCHAR(16) NOT NULL CHECK (provider IN ('github', 'google')),
  subject VARCHAR(255) NOT NULL,
  email VARCHAR(255) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT uq_oauth_subject UNIQUE (provider, subject),
  CONSTRAINT uq_oauth_user_provider UNIQUE (user_id, provider)
);

-- +goose Down
DROP TABLE oauth_identities;
