-- +migrate Up
ALTER TABLE urls ADD COLUMN IF NOT EXISTS is_deleted BOOLEAN DEFAULT FALSE;
CREATE INDEX IF NOT EXISTS idx_is_deleted ON urls(is_deleted);

-- +migrate Down
DROP INDEX IF EXISTS idx_is_deleted;
ALTER TABLE urls DROP COLUMN IF EXISTS is_deleted;
