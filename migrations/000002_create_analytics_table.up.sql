-- Create analytics table for tracking URL access
CREATE TABLE analytics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    short_code VARCHAR(50) NOT NULL,
    accessed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    ip_address INET,
    user_agent TEXT,
    referer TEXT,
    country_code VARCHAR(2),
    city VARCHAR(100),
    device_type VARCHAR(50)
);

-- Create indexes for analytics queries
CREATE INDEX idx_analytics_short_code ON analytics(short_code);
CREATE INDEX idx_analytics_accessed_at ON analytics(accessed_at);
CREATE INDEX idx_analytics_short_code_accessed ON analytics(short_code, accessed_at DESC);
CREATE INDEX idx_analytics_country_code ON analytics(country_code) WHERE country_code IS NOT NULL;
CREATE INDEX idx_analytics_device_type ON analytics(device_type) WHERE device_type IS NOT NULL;

-- Create partial indexes for common analytics queries
CREATE INDEX idx_analytics_recent ON analytics(accessed_at DESC) WHERE accessed_at > NOW() - INTERVAL '30 days';
CREATE INDEX idx_analytics_ip_short_code ON analytics(ip_address, short_code) WHERE ip_address IS NOT NULL;

-- Add foreign key constraint to urls table
ALTER TABLE analytics ADD CONSTRAINT fk_analytics_short_code 
    FOREIGN KEY (short_code) REFERENCES urls(short_code) ON DELETE CASCADE;

-- Add constraints
ALTER TABLE analytics ADD CONSTRAINT chk_analytics_country_code 
    CHECK (country_code IS NULL OR length(country_code) = 2);
ALTER TABLE analytics ADD CONSTRAINT chk_analytics_city_length 
    CHECK (city IS NULL OR length(city) <= 100);
ALTER TABLE analytics ADD CONSTRAINT chk_analytics_device_type_length 
    CHECK (device_type IS NULL OR length(device_type) <= 50);

-- Create partitioning for analytics table by month (for better performance with large datasets)
-- This is optional but recommended for high-traffic scenarios

-- Function to create monthly partitions
CREATE OR REPLACE FUNCTION create_monthly_partition(table_name text, start_date date)
RETURNS void AS $$
DECLARE
    partition_name text;
    start_month text;
    end_date date;
BEGIN
    start_month := to_char(start_date, 'YYYY_MM');
    partition_name := table_name || '_' || start_month;
    end_date := start_date + interval '1 month';
    
    EXECUTE format('CREATE TABLE IF NOT EXISTS %I PARTITION OF %I 
                    FOR VALUES FROM (%L) TO (%L)',
                   partition_name, table_name, start_date, end_date);
    
    -- Create indexes on partition
    EXECUTE format('CREATE INDEX IF NOT EXISTS %I ON %I(short_code, accessed_at DESC)',
                   'idx_' || partition_name || '_short_code_accessed', partition_name);
END;
$$ LANGUAGE plpgsql;

-- Convert analytics to partitioned table (commented out for now, can be enabled later)
-- ALTER TABLE analytics RENAME TO analytics_old;
-- CREATE TABLE analytics (LIKE analytics_old INCLUDING ALL) PARTITION BY RANGE (accessed_at);
-- 
-- -- Create initial partitions for current and next few months
-- SELECT create_monthly_partition('analytics', date_trunc('month', CURRENT_DATE));
-- SELECT create_monthly_partition('analytics', date_trunc('month', CURRENT_DATE + interval '1 month'));
-- SELECT create_monthly_partition('analytics', date_trunc('month', CURRENT_DATE + interval '2 months'));
-- 
-- -- Migrate data
-- INSERT INTO analytics SELECT * FROM analytics_old;
-- DROP TABLE analytics_old;
