-- Create extension for UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create urls table
CREATE TABLE urls (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    short_code VARCHAR(50) NOT NULL UNIQUE,
    original_url TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    click_count BIGINT DEFAULT 0,
    last_accessed_at TIMESTAMP WITH TIME ZONE,
    custom_alias VARCHAR(100),
    user_id VARCHAR(255),
    expires_at TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN DEFAULT TRUE
);

-- Create indexes for performance
CREATE INDEX idx_urls_short_code ON urls(short_code);
CREATE INDEX idx_urls_original_url ON urls(original_url);
CREATE INDEX idx_urls_user_id ON urls(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX idx_urls_created_at ON urls(created_at);
CREATE INDEX idx_urls_click_count ON urls(click_count);
CREATE INDEX idx_urls_expires_at ON urls(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX idx_urls_is_active ON urls(is_active);

-- Create composite indexes for common queries
CREATE INDEX idx_urls_user_created ON urls(user_id, created_at DESC) WHERE user_id IS NOT NULL;
CREATE INDEX idx_urls_active_expires ON urls(is_active, expires_at) WHERE expires_at IS NOT NULL;

-- Create unique constraint for custom aliases
CREATE UNIQUE INDEX idx_urls_custom_alias ON urls(custom_alias) WHERE custom_alias IS NOT NULL;

-- Add constraints
ALTER TABLE urls ADD CONSTRAINT chk_urls_short_code_length CHECK (length(short_code) >= 4 AND length(short_code) <= 50);
ALTER TABLE urls ADD CONSTRAINT chk_urls_original_url_length CHECK (length(original_url) >= 10 AND length(original_url) <= 4096);
ALTER TABLE urls ADD CONSTRAINT chk_urls_custom_alias_length CHECK (custom_alias IS NULL OR (length(custom_alias) >= 3 AND length(custom_alias) <= 100));
ALTER TABLE urls ADD CONSTRAINT chk_urls_click_count_positive CHECK (click_count >= 0);

-- Create function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create trigger to automatically update updated_at
CREATE TRIGGER update_urls_updated_at 
    BEFORE UPDATE ON urls 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();
