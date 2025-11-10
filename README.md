# URL Shortener Service

A production-ready URL shortener service built with Go, gRPC, PostgreSQL, and Redis.

## Features

- 🚀 High-performance gRPC and REST APIs
- 🔗 URL shortening with custom aliases
- 🔐 JWT and API key authentication
- 📊 Click analytics and statistics
- ⚡ Redis caching for fast redirects
- 🛡️ Rate limiting and security
- 🐳 Docker containerization
- 📈 Prometheus metrics
- 🔍 Structured logging
- ✅ Comprehensive testing
- 🔄 Complete CI/CD pipeline with GitHub Actions
- 🔒 Automated security scanning and vulnerability detection

## Architecture

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   gRPC API  │───▶│   Service   │───▶│ Repository  │
│   Handler   │    │   Layer     │    │   Layer     │
└─────────────┘    └─────────────┘    └─────────────┘
                                              │
                   ┌─────────────┐    ┌─────────────┐
                   │    Redis    │    │ PostgreSQL  │
                   │   (Cache)   │    │ (Database)  │
                   └─────────────┘    └─────────────┘
```

## Tech Stack

- **Language**: Go 1.21+
- **API**: gRPC with Protocol Buffers
- **Database**: PostgreSQL 15+
- **Cache**: Redis 7+
- **Containerization**: Docker & Docker Compose
- **CI/CD**: GitHub Actions
- **Monitoring**: Prometheus & Grafana
- **Logging**: Structured JSON logging with Zap

## Project Structure

```
url-shortener/
├── cmd/server/              # Application entrypoint
├── internal/                # Private application code
│   ├── config/             # Configuration management
│   ├── handler/grpc/       # gRPC handlers
│   ├── service/            # Business logic
│   ├── repository/         # Data access layer
│   ├── models/             # Domain models
│   ├── middleware/         # gRPC middleware
│   └── utils/              # Internal utilities
├── pkg/                    # Public packages
│   ├── shortener/          # URL shortening algorithms
│   ├── validator/          # Input validation
│   └── ratelimiter/        # Rate limiting utilities
├── proto/                  # Protocol buffer definitions
├── migrations/             # Database migrations
├── scripts/                # Build and deployment scripts
├── tests/                  # Test files
├── docker/                 # Docker configurations
└── docs/                   # Documentation
```

## Quick Start

### Prerequisites

- Go 1.21+
- Docker & Docker Compose
- PostgreSQL 15+
- Redis 7+

### Installation

1. Clone the repository:
```bash
git clone https://github.com/rajweepmondal/url-shortener.git
cd url-shortener
```

2. Copy environment file:
```bash
cp .env.example .env
```

3. Install dependencies:
```bash
go mod tidy
```

### Development

#### Option 1: Local Development
1. Start services with Docker Compose:
```bash
docker-compose up -d postgres redis
```

2. Run database migrations:
```bash
make migrate-up
```

3. Start the server:
```bash
go run cmd/server/main.go
```

#### Option 2: Full Docker Development
1. Start all services with Docker Compose:
```bash
docker-compose up --build
```

This will start PostgreSQL, Redis, and the application in containers.

### Production Deployment

#### Using Docker Compose (Recommended)
1. Copy production environment file:
```bash
cp .env.prod.example .env.prod
# Edit .env.prod with your production values
```

2. Start production services:
```bash
docker-compose -f docker-compose.prod.yml up -d
```

#### Using Pre-built Image
```bash
docker run -p 8080:8080 \
  -e DATABASE_POSTGRES_URL="your-db-url" \
  -e REDIS_URL="your-redis-url" \
  rajdweep1/url-shortener:latest
```

### Testing

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run integration tests
make test-integration
```

## Docker Commands

### Building Images
```bash
# Build Docker image
make docker-build

# Build and push to registry
make docker-build-push

# Build specific version
./scripts/docker-build.sh v1.0.0

# Build and push specific version
./scripts/docker-build.sh v1.0.0 --push
```

### Running with Docker
```bash
# Development environment
make docker-run

# Production environment
make docker-run-prod

# View logs
make docker-logs

# Stop services
make docker-down
```

### Manual Docker Commands
```bash
# Build image manually
docker build -t url-shortener:latest .

# Run single container
docker run -p 8080:8080 url-shortener:latest

# Run with environment variables
docker run -p 8080:8080 \
  -e DATABASE_POSTGRES_URL="postgres://user:pass@host:5432/db" \
  -e REDIS_URL="redis://host:6379/0" \
  url-shortener:latest
```

## Configuration

The application uses environment variables for configuration. See `.env.example` for all available options.

Key configuration sections:
- **Server**: Port, timeouts, message sizes
- **Database**: Connection settings, pool configuration
- **Redis**: Connection settings, pool configuration
- **Application**: URL length, rate limits, cache TTL
- **Authentication**: JWT secrets, API keys, auth settings
- **Logging**: Level and format

## Authentication

The service supports both JWT tokens and API keys for authentication. See [docs/AUTHENTICATION.md](docs/AUTHENTICATION.md) for detailed documentation.

### Quick Start

1. **Login to get JWT token:**
```bash
curl -X POST http://localhost:8081/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "admin"}'
```

2. **Use JWT token for API calls:**
```bash
curl -X POST http://localhost:8081/api/v1/urls \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{"original_url": "https://example.com"}'
```

3. **Test authentication system:**
```bash
./scripts/test-auth.sh
```

## API Documentation

The service exposes both gRPC and REST APIs. See `proto/` directory for Protocol Buffer definitions.

### Main Services

- `ShortenURL` - Create shortened URLs
- `GetOriginalURL` - Retrieve original URLs for redirection
- `GetURLStats` - Get click statistics
- `ListURLs` - List URLs with pagination
- `GetAnalytics` - Get detailed analytics

### Authentication Endpoints

- `POST /api/v1/auth/login` - User login
- `POST /api/v1/auth/validate` - Token validation
- `GET /api/v1/auth/profile` - Get user profile
- `POST /api/v1/auth/api-keys` - Create API key

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## CI/CD Pipeline

### 🔄 Automated Workflows
- **Continuous Integration**: Code quality, testing, security scanning
- **Continuous Deployment**: Automated staging and production deployments
- **Security Pipeline**: Daily vulnerability scans, secret detection
- **Release Management**: Automated releases with multi-platform binaries
- **Dependency Updates**: Automated updates via Dependabot

### 🛡️ Quality Gates
- **Code Quality**: 30+ linters with golangci-lint
- **Security**: Multiple scanners (Trivy, Snyk, gosec, CodeQL)
- **Testing**: Unit, integration, and performance tests
- **Coverage**: Automated coverage reporting to Codecov
- **Performance**: Automated performance validation

### 📋 Pipeline Status
- ✅ **CI Pipeline**: Comprehensive testing and quality checks
- ✅ **Security Scanning**: Daily automated vulnerability detection
- ✅ **Docker Publishing**: Multi-architecture image builds
- ✅ **Release Automation**: Tag-based releases with binaries
- ✅ **Dependency Management**: Automated security updates

For detailed CI/CD documentation, see [docs/CICD.md](docs/CICD.md).

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Performance & Testing

### 🚀 Performance Results
All critical performance requirements **exceeded**:

| Requirement | Target | Actual Result | Status |
|-------------|--------|---------------|---------|
| Redirect Performance | < 100ms | **0-20ms** | ✅ **5x Better** |
| Daily URL Capacity | 10,000 URLs/day | **449,280 URLs/day** | ✅ **44x Better** |
| Rate Limiting | Basic protection | **100 req/min sliding window** | ✅ **Enterprise Grade** |

### 🧪 Test Coverage
- **Authentication Tests**: 175/175 passed (100%)
- **URL Creation Edge Cases**: 65/65 passed (100%)
- **Redirection Edge Cases**: 32/32 passed (100%)
- **Performance Tests**: All requirements exceeded
- **Rate Limiting Tests**: Working correctly

## Quick API Examples

### Create Short URL
```bash
curl -X POST http://localhost:8081/api/v1/urls \
  -H "Content-Type: application/json" \
  -d '{"original_url": "https://www.google.com"}'
```

### Access Short URL
```bash
curl -L http://localhost:8081/{short_code}
```

### Get Analytics
```bash
curl http://localhost:8081/api/v1/urls/{short_code}/analytics
```

For detailed API examples, see [docs/API_EXAMPLES.md](docs/API_EXAMPLES.md)
