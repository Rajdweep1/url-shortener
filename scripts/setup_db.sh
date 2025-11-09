#!/bin/bash

# Database setup script for URL Shortener
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Database configuration
DB_NAME="url_shortener"
TEST_DB_NAME="url_shortener_test"
DB_USER="postgres"
DB_PASSWORD="postgres"

echo -e "${BLUE}[INFO]${NC} Setting up PostgreSQL databases for URL Shortener..."

# Function to run SQL command
run_sql() {
    local sql="$1"
    local db="${2:-postgres}"
    echo -e "${BLUE}[INFO]${NC} Executing: $sql"
    psql -d "$db" -c "$sql"
}

# Create user if it doesn't exist
echo -e "${BLUE}[INFO]${NC} Creating database user..."
psql -d postgres -c "DO \$\$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_user WHERE usename = '$DB_USER') THEN
        CREATE USER $DB_USER WITH PASSWORD '$DB_PASSWORD';
    END IF;
END
\$\$;" || echo -e "${YELLOW}[WARN]${NC} User might already exist"

# Grant necessary privileges to user
echo -e "${BLUE}[INFO]${NC} Granting privileges to user..."
psql -d postgres -c "ALTER USER $DB_USER CREATEDB;" || true
psql -d postgres -c "ALTER USER $DB_USER WITH SUPERUSER;" || true

# Create main database
echo -e "${BLUE}[INFO]${NC} Creating main database..."
psql -d postgres -c "DROP DATABASE IF EXISTS $DB_NAME;" || true
psql -d postgres -c "CREATE DATABASE $DB_NAME OWNER $DB_USER;"

# Create test database
echo -e "${BLUE}[INFO]${NC} Creating test database..."
psql -d postgres -c "DROP DATABASE IF EXISTS $TEST_DB_NAME;" || true
psql -d postgres -c "CREATE DATABASE $TEST_DB_NAME OWNER $DB_USER;"

# Grant all privileges
psql -d postgres -c "GRANT ALL PRIVILEGES ON DATABASE $DB_NAME TO $DB_USER;"
psql -d postgres -c "GRANT ALL PRIVILEGES ON DATABASE $TEST_DB_NAME TO $DB_USER;"

echo -e "${GREEN}[SUCCESS]${NC} Databases created successfully!"

# Create tables in main database
echo -e "${BLUE}[INFO]${NC} Creating tables in main database..."
psql -d "$DB_NAME" -c "
-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";

-- URLs table
CREATE TABLE IF NOT EXISTS urls (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    short_code VARCHAR(50) UNIQUE NOT NULL,
    original_url TEXT NOT NULL,
    custom_alias VARCHAR(100),
    user_id VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE,
    last_accessed_at TIMESTAMP WITH TIME ZONE,
    click_count BIGINT DEFAULT 0,
    is_active BOOLEAN DEFAULT true
);

-- Analytics table
CREATE TABLE IF NOT EXISTS analytics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    short_code VARCHAR(50) NOT NULL,
    accessed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    ip_address INET,
    user_agent TEXT,
    referer TEXT,
    country_code VARCHAR(2),
    city VARCHAR(100),
    device_type VARCHAR(50),
    FOREIGN KEY (short_code) REFERENCES urls(short_code) ON DELETE CASCADE
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_urls_short_code ON urls(short_code);
CREATE INDEX IF NOT EXISTS idx_urls_user_id ON urls(user_id);
CREATE INDEX IF NOT EXISTS idx_urls_created_at ON urls(created_at);
CREATE INDEX IF NOT EXISTS idx_urls_expires_at ON urls(expires_at);
CREATE INDEX IF NOT EXISTS idx_analytics_short_code ON analytics(short_code);
CREATE INDEX IF NOT EXISTS idx_analytics_accessed_at ON analytics(accessed_at);

-- Grant permissions
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO $DB_USER;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO $DB_USER;
"

# Create tables in test database
echo -e "${BLUE}[INFO]${NC} Creating tables in test database..."
psql -d "$TEST_DB_NAME" -c "
-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";

-- URLs table
CREATE TABLE IF NOT EXISTS urls (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    short_code VARCHAR(50) UNIQUE NOT NULL,
    original_url TEXT NOT NULL,
    custom_alias VARCHAR(100),
    user_id VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE,
    last_accessed_at TIMESTAMP WITH TIME ZONE,
    click_count BIGINT DEFAULT 0,
    is_active BOOLEAN DEFAULT true
);

-- Analytics table
CREATE TABLE IF NOT EXISTS analytics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    short_code VARCHAR(50) NOT NULL,
    accessed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    ip_address INET,
    user_agent TEXT,
    referer TEXT,
    country_code VARCHAR(2),
    city VARCHAR(100),
    device_type VARCHAR(50),
    FOREIGN KEY (short_code) REFERENCES urls(short_code) ON DELETE CASCADE
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_urls_short_code ON urls(short_code);
CREATE INDEX IF NOT EXISTS idx_urls_user_id ON urls(user_id);
CREATE INDEX IF NOT EXISTS idx_urls_created_at ON urls(created_at);
CREATE INDEX IF NOT EXISTS idx_urls_expires_at ON urls(expires_at);
CREATE INDEX IF NOT EXISTS idx_analytics_short_code ON analytics(short_code);
CREATE INDEX IF NOT EXISTS idx_analytics_accessed_at ON analytics(accessed_at);

-- Grant permissions
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO $DB_USER;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO $DB_USER;
"

echo -e "${GREEN}[SUCCESS]${NC} Database setup completed!"
echo -e "${BLUE}[INFO]${NC} Database connection details:"
echo -e "  Main DB: postgresql://$DB_USER:$DB_PASSWORD@localhost:5432/$DB_NAME"
echo -e "  Test DB: postgresql://$DB_USER:$DB_PASSWORD@localhost:5432/$TEST_DB_NAME"
