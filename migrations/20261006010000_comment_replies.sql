-- +goose Up
ALTER TABLE article_comments ADD COLUMN reply_to_id BIGINT REFERENCES article_comments(id) ON DELETE SET NULL;
CREATE INDEX idx_comments_reply_to ON article_comments(reply_to_id, created_at ASC, id ASC) WHERE status = 'approved';

-- +goose Down
ALTER TABLE article_comments DROP COLUMN reply_to_id;
