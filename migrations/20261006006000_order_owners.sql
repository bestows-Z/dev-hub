-- +goose Up
ALTER TABLE orders ADD COLUMN user_id BIGINT REFERENCES users(id);
UPDATE orders SET user_id = users.id FROM users WHERE lower(orders.email) = lower(users.email);
CREATE INDEX idx_orders_user_created ON orders (user_id, created_at DESC) WHERE user_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_orders_user_created;
ALTER TABLE orders DROP COLUMN IF EXISTS user_id;
