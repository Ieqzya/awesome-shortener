-- +migrate Up
CREATE UNIQUE INDEX IF NOT EXISTS idx_unique_original_url ON urls(original_url);

-- +migrate Down
DROP INDEX IF EXISTS idx_unique_original_url;
