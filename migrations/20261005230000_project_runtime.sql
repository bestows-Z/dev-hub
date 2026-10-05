-- +goose Up
ALTER TABLE projects
  ADD COLUMN backend_url TEXT NOT NULL DEFAULT '',
  ADD COLUMN runtime_status VARCHAR(16) NOT NULL DEFAULT 'stopped' CHECK (runtime_status IN ('stopped', 'running')),
  ADD COLUMN runtime_frontend_port INT NOT NULL DEFAULT 0,
  ADD COLUMN runtime_backend_port INT NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE projects
  DROP COLUMN IF EXISTS runtime_backend_port,
  DROP COLUMN IF EXISTS runtime_frontend_port,
  DROP COLUMN IF EXISTS runtime_status,
  DROP COLUMN IF EXISTS backend_url;
