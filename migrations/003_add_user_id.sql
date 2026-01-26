-- +migrate Up
ALTER TABLE urls ADD COLUMN IF NOT EXISTS user_id VARCHAR(255);
CREATE INDEX IF NOT EXISTS idx_user_id ON urls(user_id);

-- +migrate Down
DROP INDEX IF EXISTS idx_user_id;
ALTER TABLE urls DROP COLUMN IF EXISTS user_id;
