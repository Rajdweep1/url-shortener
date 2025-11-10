#!/bin/bash

# Database cleanup script for testing
# This script clears all data from the URLs table to ensure clean test runs

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🧹 Cleaning up database for fresh test run${NC}"

# Database connection details
CONTAINER_NAME="url-shortener-postgres"
DB_NAME="url_shortener"
DB_USER="user"

# Wait for database container to be ready
echo -e "${YELLOW}⏳ Waiting for database container to be ready...${NC}"
for i in {1..30}; do
    if docker exec $CONTAINER_NAME pg_isready -U $DB_USER -d $DB_NAME >/dev/null 2>&1; then
        echo -e "${GREEN}✓ Database container is ready${NC}"
        break
    fi
    if [ $i -eq 30 ]; then
        echo -e "${RED}❌ Database container is not ready after 30 seconds${NC}"
        exit 1
    fi
    sleep 1
done

# Clear the URLs table
echo -e "${YELLOW}🗑️  Clearing URLs table...${NC}"
docker exec $CONTAINER_NAME psql -U $DB_USER -d $DB_NAME -c "TRUNCATE TABLE urls RESTART IDENTITY CASCADE;" >/dev/null 2>&1

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Database cleaned successfully${NC}"
else
    echo -e "${RED}❌ Failed to clean database${NC}"
    exit 1
fi

echo -e "${BLUE}🎉 Database is now clean and ready for testing${NC}"
