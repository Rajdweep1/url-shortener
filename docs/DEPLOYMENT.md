# Deployment Guide

This guide covers different deployment options for the URL Shortener service.

## Prerequisites

- Docker and Docker Compose installed
- PostgreSQL 15+ and Redis 7+ (if not using Docker)
- Domain name and SSL certificate (for production)

## Quick Start with Docker Compose

The fastest way to get the service running:

```bash
# Clone the repository
git clone https://github.com/rajdweep1/url-shortener.git
cd url-shortener

# Start all services
docker-compose up --build
```

The service will be available at `http://localhost:8080`.

## Production Deployment

### 1. Using Docker Compose (Recommended)

1. **Prepare environment file:**
```bash
cp .env.prod.example .env.prod
# Edit .env.prod with your production values
```

2. **Update configuration:**
```bash
# Edit .env.prod
DATABASE_POSTGRES_URL=postgresql://user:secure_password@postgres:5432/url_shortener?sslmode=require
APP_BASE_URL=https://yourdomain.com
LOG_LEVEL=warn
```

3. **Deploy:**
```bash
docker-compose -f docker-compose.prod.yml up -d
```

4. **Verify deployment:**
```bash
docker-compose -f docker-compose.prod.yml ps
docker-compose -f docker-compose.prod.yml logs app
```

### 2. Using Pre-built Docker Image

```bash
docker run -d \
  --name url-shortener \
  -p 8080:8080 \
  -e DATABASE_POSTGRES_URL="your-postgres-url" \
  -e REDIS_URL="your-redis-url" \
  -e APP_BASE_URL="https://yourdomain.com" \
  --restart unless-stopped \
  rajdweep1/url-shortener:latest
```

### 3. Kubernetes Deployment

Create Kubernetes manifests:

```yaml
# k8s/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: url-shortener
spec:
  replicas: 3
  selector:
    matchLabels:
      app: url-shortener
  template:
    metadata:
      labels:
        app: url-shortener
    spec:
      containers:
      - name: url-shortener
        image: rajdweep1/url-shortener:latest
        ports:
        - containerPort: 8080
        env:
        - name: DATABASE_POSTGRES_URL
          valueFrom:
            secretKeyRef:
              name: url-shortener-secrets
              key: database-url
        - name: REDIS_URL
          valueFrom:
            secretKeyRef:
              name: url-shortener-secrets
              key: redis-url
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          exec:
            command:
            - nc
            - -z
            - localhost
            - "8080"
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          exec:
            command:
            - nc
            - -z
            - localhost
            - "8080"
          initialDelaySeconds: 5
          periodSeconds: 5
```

## Environment Variables

### Required Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `DATABASE_POSTGRES_URL` | PostgreSQL connection string | `postgres://user:pass@host:5432/db` |
| `REDIS_URL` | Redis connection string | `redis://host:6379/0` |
| `APP_BASE_URL` | Base URL for shortened links | `https://yourdomain.com` |

### Optional Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | `8080` | Server port |
| `APP_SHORT_CODE_LENGTH` | `7` | Length of generated short codes |
| `APP_RATE_LIMIT` | `100` | Requests per minute per IP |
| `LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |

## SSL/TLS Configuration

### Using Reverse Proxy (Recommended)

Use nginx or Traefik as a reverse proxy:

```nginx
# nginx.conf
server {
    listen 443 ssl http2;
    server_name yourdomain.com;
    
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    
    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## Monitoring and Logging

### Health Checks

The service provides health check endpoints:

```bash
# Check if service is running
curl http://localhost:8080/health

# Check database connectivity
curl http://localhost:8080/health/db
```

### Logging

Logs are written to stdout in JSON format. Configure log aggregation:

```bash
# View logs
docker-compose logs -f app

# With log driver
docker run --log-driver=syslog rajdweep1/url-shortener:latest
```

## Scaling

### Horizontal Scaling

The service is stateless and can be scaled horizontally:

```bash
# Scale with Docker Compose
docker-compose up --scale app=3

# Scale with Kubernetes
kubectl scale deployment url-shortener --replicas=5
```

### Database Scaling

- Use PostgreSQL read replicas for read-heavy workloads
- Implement connection pooling (PgBouncer)
- Consider database sharding for very high scale

### Redis Scaling

- Use Redis Cluster for high availability
- Implement Redis Sentinel for automatic failover
- Consider Redis partitioning for large datasets

## Backup and Recovery

### Database Backup

```bash
# Backup PostgreSQL
docker exec postgres pg_dump -U user url_shortener > backup.sql

# Restore PostgreSQL
docker exec -i postgres psql -U user url_shortener < backup.sql
```

### Redis Backup

```bash
# Backup Redis
docker exec redis redis-cli BGSAVE
docker cp redis:/data/dump.rdb ./redis-backup.rdb
```

## Troubleshooting

### Common Issues

1. **Service won't start:**
   - Check environment variables
   - Verify database connectivity
   - Check port availability

2. **Database connection errors:**
   - Verify PostgreSQL is running
   - Check connection string format
   - Ensure database exists

3. **Redis connection errors:**
   - Verify Redis is running
   - Check Redis URL format
   - Test Redis connectivity

### Debug Commands

```bash
# Check service status
docker-compose ps

# View logs
docker-compose logs app

# Connect to database
docker exec -it postgres psql -U user url_shortener

# Connect to Redis
docker exec -it redis redis-cli

# Test gRPC service
grpcurl -plaintext localhost:8080 list
```

## Performance Tuning

### Database Optimization

- Create appropriate indexes
- Tune PostgreSQL configuration
- Use connection pooling
- Monitor query performance

### Redis Optimization

- Configure appropriate memory limits
- Use appropriate eviction policies
- Monitor memory usage
- Optimize key expiration

### Application Optimization

- Tune Go runtime settings
- Configure appropriate timeouts
- Monitor goroutine usage
- Profile memory usage

## Security Considerations

1. **Use strong passwords** for database connections
2. **Enable SSL/TLS** for all connections
3. **Implement rate limiting** to prevent abuse
4. **Use secrets management** for sensitive data
5. **Regular security updates** for base images
6. **Network segmentation** in production
7. **Monitor for suspicious activity**
