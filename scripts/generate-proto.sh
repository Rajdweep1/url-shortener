#!/bin/bash

# Script to generate Protocol Buffer files
# Usage: ./scripts/generate-proto.sh

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Generating Protocol Buffer files...${NC}"

# Check if protoc is installed
if ! command -v protoc &> /dev/null; then
    echo -e "${RED}Error: protoc is not installed${NC}"
    echo "Please install Protocol Buffers compiler:"
    echo "  macOS: brew install protobuf"
    echo "  Ubuntu: sudo apt-get install protobuf-compiler"
    exit 1
fi

# Check if Go plugins are installed
GOPATH=$(go env GOPATH)
if [ ! -f "$GOPATH/bin/protoc-gen-go" ]; then
    echo -e "${RED}Error: protoc-gen-go is not installed${NC}"
    echo "Please install: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest"
    exit 1
fi

if [ ! -f "$GOPATH/bin/protoc-gen-go-grpc" ]; then
    echo -e "${RED}Error: protoc-gen-go-grpc is not installed${NC}"
    echo "Please install: go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest"
    exit 1
fi

# Create output directory
mkdir -p proto/gen/go/url_shortener/v1

# Generate Go code from proto files
protoc \
    --go_out=. \
    --go-grpc_out=. \
    --plugin=protoc-gen-go="$GOPATH/bin/protoc-gen-go" \
    --plugin=protoc-gen-go-grpc="$GOPATH/bin/protoc-gen-go-grpc" \
    proto/*.proto

# Clean up any incorrectly placed files
if [ -d "github.com" ]; then
    echo -e "${YELLOW}Moving generated files to correct location...${NC}"
    mv github.com/rajweepmondal/url-shortener/proto/gen/go/url_shortener/v1/* proto/gen/go/url_shortener/v1/ 2>/dev/null || true
    rm -rf github.com
fi

echo -e "${GREEN}✓ Protocol Buffer files generated successfully${NC}"
echo "Generated files:"
ls -la proto/gen/go/url_shortener/v1/

echo -e "${GREEN}✓ Done!${NC}"
