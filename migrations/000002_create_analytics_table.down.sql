-- Drop function
DROP FUNCTION IF EXISTS create_monthly_partition(text, date);

-- Drop indexes
DROP INDEX IF EXISTS idx_analytics_ip_short_code;
DROP INDEX IF EXISTS idx_analytics_recent;
DROP INDEX IF EXISTS idx_analytics_device_type;
DROP INDEX IF EXISTS idx_analytics_country_code;
DROP INDEX IF EXISTS idx_analytics_short_code_accessed;
DROP INDEX IF EXISTS idx_analytics_accessed_at;
DROP INDEX IF EXISTS idx_analytics_short_code;

-- Drop table
DROP TABLE IF EXISTS analytics;
