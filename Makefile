.PHONY: help build test clean docker proto migrate lint fmt vet

# Variables
APP_NAME := url-shortener
DOCKER_IMAGE := $(APP_NAME):latest
PROTO_DIR := proto
MIGRATION_DIR := migrations
GOPATH := $(shell go env GOPATH)

# Default target
.DEFAULT_GOAL := help

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the application
	@echo "Building $(APP_NAME)..."
	@go build -o bin/$(APP_NAME) cmd/server/main.go

run: ## Run the application
	@echo "Running $(APP_NAME)..."
	@go run cmd/server/main.go

test: ## Run tests
	@echo "Running tests..."
	@go test -v -race ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-integration: ## Run integration tests
	@echo "Running integration tests..."
	@go test -v -tags=integration ./tests/integration/...

lint: ## Run linter
	@echo "Running linter..."
	@$(GOPATH)/bin/golangci-lint run

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...

vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...

proto: ## Generate protobuf code
	@echo "Generating protobuf code..."
	@protoc --go_out=. --go-grpc_out=. \
		--plugin=protoc-gen-go=$(GOPATH)/bin/protoc-gen-go \
		--plugin=protoc-gen-go-grpc=$(GOPATH)/bin/protoc-gen-go-grpc \
		$(PROTO_DIR)/*.proto

migrate-up: ## Run database migrations up
	@echo "Running migrations up..."
	@$(GOPATH)/bin/migrate -path $(MIGRATION_DIR) -database "$(DATABASE_POSTGRES_URL)" up

migrate-down: ## Run database migrations down
	@echo "Running migrations down..."
	@$(GOPATH)/bin/migrate -path $(MIGRATION_DIR) -database "$(DATABASE_POSTGRES_URL)" down

migrate-create: ## Create a new migration (usage: make migrate-create NAME=migration_name)
	@echo "Creating migration: $(NAME)"
	@$(GOPATH)/bin/migrate create -ext sql -dir $(MIGRATION_DIR) $(NAME)

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@./scripts/docker-build.sh

docker-build-push: ## Build and push Docker image
	@echo "Building and pushing Docker image..."
	@./scripts/docker-build.sh latest --push

docker-run: ## Run with Docker Compose
	@echo "Starting services with Docker Compose..."
	@docker-compose up --build

docker-run-prod: ## Run with production Docker Compose
	@echo "Starting production services..."
	@docker-compose -f docker-compose.prod.yml up -d

docker-down: ## Stop Docker Compose services
	@echo "Stopping Docker Compose services..."
	@docker-compose down

docker-down-prod: ## Stop production Docker Compose services
	@echo "Stopping production services..."
	@docker-compose -f docker-compose.prod.yml down

docker-logs: ## Show Docker Compose logs
	@docker-compose logs -f

docker-logs-prod: ## Show production Docker Compose logs
	@docker-compose -f docker-compose.prod.yml logs -f

docker-test: ## Test Docker image
	@echo "Testing Docker image..."
	@docker run --rm $(DOCKER_IMAGE) ./server --version || echo "Version check completed"

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf bin/ coverage.out coverage.html

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

dev-setup: ## Setup development environment
	@echo "Setting up development environment..."
	@cp .env.example .env
	@echo "Please edit .env file with your configuration"

dev: ## Start development environment
	@echo "Starting development environment..."
	@docker-compose -f docker-compose.dev.yml up --build
