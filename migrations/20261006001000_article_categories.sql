-- +goose Up
ALTER TABLE articles ADD COLUMN category VARCHAR(16) NOT NULL DEFAULT 'tech'
  CHECK (category IN ('tech', 'travel', 'essay', 'record'));
CREATE INDEX idx_articles_category_public ON articles (category, published_at DESC, id DESC)
  WHERE status = 'published';
CREATE INDEX idx_articles_tags_public ON articles USING GIN (tags)
  WHERE status = 'published';

-- +goose Down
DROP INDEX IF EXISTS idx_articles_tags_public;
DROP INDEX IF EXISTS idx_articles_category_public;
ALTER TABLE articles DROP COLUMN IF EXISTS category;
