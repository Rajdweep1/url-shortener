# CI/CD Pipeline Documentation

## Overview

This project uses GitHub Actions for comprehensive CI/CD automation with multiple workflows covering testing, security, building, and deployment.

## Workflows

### 1. CI Pipeline (`.github/workflows/ci.yml`)

**Triggers:**
- Push to `main` or `develop` branches
- Pull requests to `main` or `develop` branches

**Jobs:**
1. **Code Quality & Security**
   - Go vet, staticcheck, gosec
   - golangci-lint with comprehensive rules
   - Security vulnerability scanning

2. **Unit Tests**
   - Full test suite with race detection
   - Code coverage reporting to Codecov
   - Coverage artifacts generation

3. **Integration Tests**
   - PostgreSQL and Redis services
   - Database migration testing
   - End-to-end API testing

4. **Docker Build & Test**
   - Multi-stage Docker build
   - Container functionality testing
   - Image security scanning

5. **Performance Tests**
   - Redirect performance validation (<100ms)
   - Throughput testing (10K+ URLs/day)
   - Rate limiting verification

6. **Build Info Generation**
   - Version tagging
   - Build metadata
   - Deployment summaries

### 2. CD Pipeline (`.github/workflows/cd.yml`)

**Triggers:**
- Push to `main` branch
- Git tags starting with `v*`
- Successful CI pipeline completion

**Jobs:**
1. **Build and Push Docker Images**
   - Multi-architecture builds (amd64, arm64)
   - Docker Hub publishing
   - SBOM (Software Bill of Materials) generation

2. **Security Scan of Published Images**
   - Trivy vulnerability scanning
   - SARIF report generation

3. **Deploy to Staging**
   - Automated staging deployment
   - Smoke tests execution
   - Deployment notifications

4. **Deploy to Production**
   - Production deployment (tag-triggered)
   - Health checks validation
   - GitHub release creation

5. **Cleanup**
   - Old image cleanup
   - Metrics updates

### 3. Security Pipeline (`.github/workflows/security.yml`)

**Triggers:**
- Daily schedule (2 AM UTC)
- Push to `main` branch
- Pull requests
- Manual dispatch

**Jobs:**
1. **Dependency Vulnerability Scan**
   - govulncheck for Go vulnerabilities
   - Nancy for dependency scanning

2. **Code Security Analysis**
   - CodeQL static analysis
   - Semgrep security rules

3. **Secret Scanning**
   - TruffleHog for secret detection
   - GitLeaks for credential scanning

4. **Container Security**
   - Trivy container scanning
   - Snyk vulnerability assessment

5. **Infrastructure Security**
   - Checkov for IaC security
   - Hadolint for Dockerfile best practices

6. **Security Reporting**
   - Consolidated security reports
   - Automated issue creation on failures

### 4. Release Pipeline (`.github/workflows/release.yml`)

**Triggers:**
- Git tags starting with `v*`
- Manual workflow dispatch

**Jobs:**
1. **Validate Release**
   - Version format validation
   - Pre-release detection

2. **Build Release Artifacts**
   - Multi-platform binary builds
   - Archive creation with checksums
   - Cross-compilation for Linux, macOS, Windows

3. **Build and Push Docker Images**
   - Release-tagged Docker images
   - Multi-architecture support

4. **Create GitHub Release**
   - Automated changelog generation
   - Asset uploads
   - Release notes creation

5. **Post-Release Tasks**
   - Documentation updates
   - Stakeholder notifications

## Configuration Files

### Dependabot (`.github/dependabot.yml`)
- **Go modules**: Weekly updates on Mondays
- **Docker**: Base image updates
- **GitHub Actions**: Workflow dependency updates
- Automatic PR creation with proper labeling

### GolangCI-Lint (`.golangci.yml`)
- **30+ linters enabled** for comprehensive code quality
- Custom rules for project-specific requirements
- Performance and security-focused checks
- Test file exclusions for appropriate linters

## Security Features

### Vulnerability Scanning
- **Daily automated scans** for dependencies and containers
- **Multiple scanners**: Trivy, Snyk, govulncheck, Nancy
- **SARIF integration** with GitHub Security tab

### Secret Detection
- **TruffleHog** for comprehensive secret scanning
- **GitLeaks** for credential detection
- **Automated alerts** on secret detection

### Code Security
- **CodeQL** static analysis
- **Semgrep** with OWASP Top 10 rules
- **gosec** for Go-specific security issues

## Performance Testing

### Automated Performance Validation
- **Redirect speed**: <100ms requirement validation
- **Throughput**: 10K+ URLs/day capacity testing
- **Rate limiting**: Proper limit enforcement
- **Load testing**: Concurrent request handling

### Performance Metrics
- Response time percentiles
- Throughput measurements
- Error rate monitoring
- Resource utilization tracking

## Deployment Strategy

### Staging Environment
- **Automatic deployment** on main branch updates
- **Smoke tests** for basic functionality
- **Integration testing** with external services

### Production Environment
- **Tag-based deployment** for controlled releases
- **Blue-green deployment** strategy support
- **Health checks** and rollback capabilities
- **Zero-downtime deployments**

## Monitoring and Alerting

### Build Monitoring
- **Slack/Teams notifications** for build failures
- **Email alerts** for security issues
- **GitHub issue creation** for critical failures

### Deployment Tracking
- **Release notes** generation
- **Deployment metrics** collection
- **Rollback procedures** documentation

## Usage

### Running CI Locally
```bash
# Install dependencies
go mod download

# Run linting
golangci-lint run

# Run tests with coverage
go test -v -race -coverprofile=coverage.out ./...

# Run security scan
gosec ./...
```

### Manual Deployment
```bash
# Create and push a release tag
git tag v1.0.0
git push origin v1.0.0

# This triggers the release pipeline automatically
```

### Environment Variables Required

#### Docker Hub
- `DOCKER_USERNAME`: Docker Hub username
- `DOCKER_PASSWORD`: Docker Hub password/token

#### Security Scanning
- `SNYK_TOKEN`: Snyk authentication token
- `CODECOV_TOKEN`: Codecov upload token

#### Notifications (Optional)
- `SLACK_WEBHOOK`: Slack notification webhook
- `TEAMS_WEBHOOK`: Microsoft Teams webhook

## Best Practices

### Branch Protection
- **Required status checks** for all CI jobs
- **Require branches to be up to date**
- **Restrict pushes** to main branch
- **Require pull request reviews**

### Security
- **Least privilege** for workflow permissions
- **Secret scanning** enabled
- **Dependency updates** automated
- **Security alerts** configured

### Performance
- **Parallel job execution** where possible
- **Caching strategies** for dependencies
- **Artifact reuse** between workflows
- **Optimized Docker builds**

## Troubleshooting

### Common Issues
1. **Test failures**: Check service dependencies
2. **Docker build failures**: Verify Dockerfile syntax
3. **Security scan failures**: Review and fix vulnerabilities
4. **Deployment failures**: Check environment configuration

### Debug Commands
```bash
# Check workflow status
gh workflow list

# View workflow runs
gh run list

# Download artifacts
gh run download <run-id>
```

## Future Enhancements

### Planned Features
- **Kubernetes deployment** support
- **Multi-cloud deployment** strategies
- **Advanced monitoring** integration
- **Automated rollback** mechanisms
- **Canary deployment** support

### Metrics and Observability
- **Prometheus metrics** collection
- **Grafana dashboards** for visualization
- **Distributed tracing** with Jaeger
- **Log aggregation** with ELK stack
