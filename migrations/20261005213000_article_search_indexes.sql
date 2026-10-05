-- +goose Up
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX idx_articles_title_trgm ON articles USING gin (title gin_trgm_ops) WHERE status = 'published';
CREATE INDEX idx_articles_excerpt_trgm ON articles USING gin (excerpt gin_trgm_ops) WHERE status = 'published';
CREATE INDEX idx_articles_body_trgm ON articles USING gin (body_md gin_trgm_ops) WHERE status = 'published';

-- +goose Down
DROP INDEX IF EXISTS idx_articles_body_trgm;
DROP INDEX IF EXISTS idx_articles_excerpt_trgm;
DROP INDEX IF EXISTS idx_articles_title_trgm;
