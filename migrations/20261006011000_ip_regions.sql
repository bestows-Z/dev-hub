-- +goose Up
ALTER TABLE users ADD COLUMN last_ip_region VARCHAR(100) NOT NULL DEFAULT '';
ALTER TABLE article_comments ADD COLUMN ip_region VARCHAR(100) NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE article_comments DROP COLUMN ip_region;
ALTER TABLE users DROP COLUMN last_ip_region;
