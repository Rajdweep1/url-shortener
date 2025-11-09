#!/bin/bash

# URL Shortener Test Runner
# This script runs different types of tests for the URL shortener service

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
RUN_UNIT=true
RUN_INTEGRATION=false
RUN_BENCHMARKS=false
VERBOSE=false
COVERAGE=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --unit-only)
            RUN_UNIT=true
            RUN_INTEGRATION=false
            RUN_BENCHMARKS=false
            shift
            ;;
        --integration)
            RUN_INTEGRATION=true
            shift
            ;;
        --benchmarks)
            RUN_BENCHMARKS=true
            shift
            ;;
        --all)
            RUN_UNIT=true
            RUN_INTEGRATION=true
            RUN_BENCHMARKS=true
            shift
            ;;
        --verbose|-v)
            VERBOSE=true
            shift
            ;;
        --coverage)
            COVERAGE=true
            shift
            ;;
        --help|-h)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --unit-only     Run only unit tests (default)"
            echo "  --integration   Run integration tests (requires databases)"
            echo "  --benchmarks    Run benchmark tests"
            echo "  --all           Run all tests"
            echo "  --verbose, -v   Verbose output"
            echo "  --coverage      Generate coverage report"
            echo "  --help, -h      Show this help message"
            echo ""
            echo "Examples:"
            echo "  $0                    # Run unit tests only"
            echo "  $0 --all --coverage   # Run all tests with coverage"
            echo "  $0 --integration      # Run integration tests"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if required tools are installed
check_dependencies() {
    print_status "Checking dependencies..."
    
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed or not in PATH"
        exit 1
    fi
    
    print_success "Go found: $(go version)"
}

# Function to run unit tests
run_unit_tests() {
    print_status "Running unit tests..."
    
    local test_args="-race"
    
    if [ "$VERBOSE" = true ]; then
        test_args="$test_args -v"
    fi
    
    if [ "$COVERAGE" = true ]; then
        test_args="$test_args -coverprofile=coverage.out -covermode=atomic"
        print_status "Coverage enabled - will generate coverage.out"
    fi
    
    # Run tests excluding integration tests
    if go test $test_args ./internal/... ./pkg/... -short; then
        print_success "Unit tests passed!"
        
        if [ "$COVERAGE" = true ]; then
            print_status "Generating coverage report..."
            go tool cover -html=coverage.out -o coverage.html
            print_success "Coverage report generated: coverage.html"
            
            # Show coverage summary
            coverage_percent=$(go tool cover -func=coverage.out | grep total | awk '{print $3}')
            print_status "Total coverage: $coverage_percent"
        fi
    else
        print_error "Unit tests failed!"
        return 1
    fi
}

# Function to check if databases are available for integration tests
check_databases() {
    print_status "Checking database availability..."
    
    # Check PostgreSQL
    if ! nc -z localhost 5432 2>/dev/null; then
        print_warning "PostgreSQL not available on localhost:5432"
        print_status "You can start PostgreSQL with Docker:"
        print_status "docker run -d --name postgres-test -e POSTGRES_PASSWORD=password -e POSTGRES_DB=url_shortener_test -p 5432:5432 postgres:15"
        return 1
    fi
    
    # Check Redis
    if ! nc -z localhost 6379 2>/dev/null; then
        print_warning "Redis not available on localhost:6379"
        print_status "You can start Redis with Docker:"
        print_status "docker run -d --name redis-test -p 6379:6379 redis:7-alpine"
        return 1
    fi
    
    print_success "Databases are available"
    return 0
}

# Function to run integration tests
run_integration_tests() {
    print_status "Running integration tests..."

    if ! check_databases; then
        print_error "Databases not available for integration tests"
        return 1
    fi

    # Load test environment variables
    if [ -f ".env.test" ]; then
        print_status "Loading test environment variables..."
        export $(grep -v '^#' .env.test | xargs)
    fi

    local test_args="-race"

    if [ "$VERBOSE" = true ]; then
        test_args="$test_args -v"
    fi
    
    # Set environment variables for integration tests
    export RUN_INTEGRATION_TESTS=true
    export TEST_DATABASE_URL="postgres://postgres:password@localhost:5432/url_shortener_test?sslmode=disable"
    export TEST_REDIS_URL="redis://localhost:6379/1"
    
    if go test $test_args ./test/...; then
        print_success "Integration tests passed!"
    else
        print_error "Integration tests failed!"
        return 1
    fi
}

# Function to run benchmark tests
run_benchmarks() {
    print_status "Running benchmark tests..."
    
    local bench_args="-bench=. -benchmem"
    
    if [ "$VERBOSE" = true ]; then
        bench_args="$bench_args -v"
    fi
    
    if go test $bench_args ./internal/... ./pkg/...; then
        print_success "Benchmarks completed!"
    else
        print_error "Benchmarks failed!"
        return 1
    fi
}

# Function to setup test databases with Docker
setup_test_databases() {
    print_status "Setting up test databases with Docker..."
    
    # Start PostgreSQL
    if ! docker ps | grep -q postgres-test; then
        print_status "Starting PostgreSQL container..."
        docker run -d --name postgres-test \
            -e POSTGRES_PASSWORD=password \
            -e POSTGRES_DB=url_shortener_test \
            -p 5432:5432 \
            postgres:15
        
        # Wait for PostgreSQL to be ready
        print_status "Waiting for PostgreSQL to be ready..."
        sleep 5
        while ! nc -z localhost 5432; do
            sleep 1
        done
    fi
    
    # Start Redis
    if ! docker ps | grep -q redis-test; then
        print_status "Starting Redis container..."
        docker run -d --name redis-test \
            -p 6379:6379 \
            redis:7-alpine
        
        # Wait for Redis to be ready
        print_status "Waiting for Redis to be ready..."
        sleep 2
        while ! nc -z localhost 6379; do
            sleep 1
        done
    fi
    
    print_success "Test databases are ready!"
}

# Main execution
main() {
    print_status "URL Shortener Test Runner"
    print_status "========================="
    
    check_dependencies
    
    # Track overall success
    overall_success=true
    
    if [ "$RUN_UNIT" = true ]; then
        if ! run_unit_tests; then
            overall_success=false
        fi
    fi
    
    if [ "$RUN_INTEGRATION" = true ]; then
        if ! run_integration_tests; then
            overall_success=false
        fi
    fi
    
    if [ "$RUN_BENCHMARKS" = true ]; then
        if ! run_benchmarks; then
            overall_success=false
        fi
    fi
    
    echo ""
    if [ "$overall_success" = true ]; then
        print_success "All tests completed successfully! 🎉"
        exit 0
    else
        print_error "Some tests failed! ❌"
        exit 1
    fi
}

# Run main function
main "$@"
