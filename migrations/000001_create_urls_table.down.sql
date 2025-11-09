-- Drop trigger
DROP TRIGGER IF EXISTS update_urls_updated_at ON urls;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop indexes
DROP INDEX IF EXISTS idx_urls_custom_alias;
DROP INDEX IF EXISTS idx_urls_active_expires;
DROP INDEX IF EXISTS idx_urls_user_created;
DROP INDEX IF EXISTS idx_urls_is_active;
DROP INDEX IF EXISTS idx_urls_expires_at;
DROP INDEX IF EXISTS idx_urls_click_count;
DROP INDEX IF EXISTS idx_urls_created_at;
DROP INDEX IF EXISTS idx_urls_user_id;
DROP INDEX IF EXISTS idx_urls_original_url;
DROP INDEX IF EXISTS idx_urls_short_code;

-- Drop table
DROP TABLE IF EXISTS urls;

-- Drop extension (only if no other tables use it)
-- DROP EXTENSION IF EXISTS "uuid-ossp";
