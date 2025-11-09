#!/bin/bash

# Docker build script for URL Shortener
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
IMAGE_NAME="rajdweep1/url-shortener"
VERSION=${1:-"latest"}
DOCKERFILE=${2:-"Dockerfile"}

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Function to check if Docker is running
check_docker() {
    if ! docker info > /dev/null 2>&1; then
        print_error "Docker is not running. Please start Docker and try again."
        exit 1
    fi
}

# Function to build Docker image
build_image() {
    print_status "Building Docker image: ${IMAGE_NAME}:${VERSION}"
    
    # Build the image
    docker build \
        --tag "${IMAGE_NAME}:${VERSION}" \
        --tag "${IMAGE_NAME}:latest" \
        --file "${DOCKERFILE}" \
        --build-arg VERSION="${VERSION}" \
        --build-arg BUILD_DATE="$(date -u +'%Y-%m-%dT%H:%M:%SZ')" \
        --build-arg VCS_REF="$(git rev-parse --short HEAD)" \
        .
    
    if [ $? -eq 0 ]; then
        print_success "Docker image built successfully!"
    else
        print_error "Failed to build Docker image"
        exit 1
    fi
}

# Function to show image info
show_image_info() {
    print_status "Image information:"
    docker images "${IMAGE_NAME}" --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}\t{{.CreatedAt}}"
    
    print_status "Image layers:"
    docker history "${IMAGE_NAME}:${VERSION}" --format "table {{.CreatedBy}}\t{{.Size}}"
}

# Function to test the image
test_image() {
    print_status "Testing the Docker image..."
    
    # Run a quick test to ensure the binary works
    docker run --rm "${IMAGE_NAME}:${VERSION}" ./server --help > /dev/null 2>&1
    
    if [ $? -eq 0 ]; then
        print_success "Image test passed!"
    else
        print_warning "Image test failed, but this might be expected if --help is not implemented"
    fi
}

# Function to push image to registry
push_image() {
    if [ "$PUSH" = "true" ]; then
        print_status "Pushing image to Docker registry..."
        
        docker push "${IMAGE_NAME}:${VERSION}"
        docker push "${IMAGE_NAME}:latest"
        
        if [ $? -eq 0 ]; then
            print_success "Image pushed successfully!"
        else
            print_error "Failed to push image"
            exit 1
        fi
    fi
}

# Main execution
main() {
    print_status "URL Shortener Docker Build Script"
    print_status "================================="
    
    # Check prerequisites
    check_docker
    
    # Build image
    build_image
    
    # Show image info
    show_image_info
    
    # Test image
    test_image
    
    # Push if requested
    push_image
    
    print_success "Build process completed!"
    print_status "To run the image: docker run -p 8080:8080 ${IMAGE_NAME}:${VERSION}"
    print_status "To run with compose: docker-compose up --build"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --push)
            PUSH="true"
            shift
            ;;
        --help)
            echo "Usage: $0 [VERSION] [DOCKERFILE] [--push] [--help]"
            echo ""
            echo "Arguments:"
            echo "  VERSION     Docker image version (default: latest)"
            echo "  DOCKERFILE  Dockerfile to use (default: Dockerfile)"
            echo ""
            echo "Options:"
            echo "  --push      Push image to registry after building"
            echo "  --help      Show this help message"
            echo ""
            echo "Examples:"
            echo "  $0                          # Build with latest tag"
            echo "  $0 v1.0.0                  # Build with v1.0.0 tag"
            echo "  $0 v1.0.0 --push           # Build and push v1.0.0"
            exit 0
            ;;
        *)
            break
            ;;
    esac
done

# Run main function
main
