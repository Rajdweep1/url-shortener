# API Testing Examples

The URL Shortener service provides **both gRPC and REST APIs** for maximum flexibility:

- **gRPC API**: Port 8080 (for high-performance applications)
- **REST API**: Port 8081 (for web applications, Postman, curl)

## Prerequisites
```bash
# For gRPC testing - Install grpcurl
brew install grpcurl  # macOS
# or
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

# For REST testing - Use curl, Postman, or any HTTP client

# Start the service
docker-compose up --build
```

## 🌐 **REST API Examples (Use with Postman!)**

### 1. 🔗 Create Shortened URL

**REST API (Postman-friendly):**
```bash
# POST http://localhost:8081/api/v1/urls
curl -X POST http://localhost:8081/api/v1/urls \
  -H "Content-Type: application/json" \
  -d '{
    "original_url": "https://www.google.com",
    "custom_alias": "google"
  }'

# Without custom alias (auto-generated)
curl -X POST http://localhost:8081/api/v1/urls \
  -H "Content-Type: application/json" \
  -d '{
    "original_url": "https://www.github.com"
  }'

# With expiration (7 days)
curl -X POST http://localhost:8081/api/v1/urls \
  -H "Content-Type: application/json" \
  -d '{
    "original_url": "https://www.example.com",
    "expires_in_days": 7
  }'
```

**Response:**
```json
{
  "url": {
    "id": "4e632693-31fc-4d85-9c71-b176e57aa690",
    "short_code": "google",
    "original_url": "https://www.google.com",
    "created_at": "2025-11-09T12:12:28.480021970Z",
    "updated_at": "2025-11-09T12:12:28.480022137Z",
    "click_count": 0,
    "custom_alias": "google",
    "is_active": true
  },
  "short_url": "http://localhost:8081/google"
}
```

### 2. 🔍 Get Original URL (Redirect)

**REST API:**
```bash
# GET http://localhost:8081/{shortCode} - Redirects to original URL
curl -L http://localhost:8081/google
# This will redirect (302) to https://www.google.com
```

### 3. ℹ️ Get URL Information

**REST API:**
```bash
# GET http://localhost:8081/api/v1/urls/{shortCode}
curl http://localhost:8081/api/v1/urls/google
```

**Response:**
```json
{
  "id": "4e632693-31fc-4d85-9c71-b176e57aa690",
  "short_code": "google",
  "original_url": "https://www.google.com",
  "created_at": "2025-11-09T12:12:28.480021970Z",
  "updated_at": "2025-11-09T12:12:28.480022137Z",
  "click_count": 5,
  "last_accessed_at": "2025-11-09T12:15:30.123456789Z",
  "custom_alias": "google",
  "is_active": true
}
```

### 4. 📋 List URLs (with pagination)

**REST API:**
```bash
# GET http://localhost:8081/api/v1/urls
curl "http://localhost:8081/api/v1/urls?page_size=10"

# Next page
curl "http://localhost:8081/api/v1/urls?page_size=10&page_token=eyJvZmZzZXQiOjEwfQ=="
```

**Response:**
```json
{
  "urls": [
    {
      "id": "4e632693-31fc-4d85-9c71-b176e57aa690",
      "short_code": "google",
      "original_url": "https://www.google.com",
      "created_at": "2025-11-09T12:12:28.480021970Z",
      "click_count": 5,
      "is_active": true
    }
  ],
  "next_page_token": "eyJvZmZzZXQiOjEwfQ=="
}
```

### 5. ✏️ Update URL

**REST API:**
```bash
# PUT http://localhost:8081/api/v1/urls/{shortCode}
curl -X PUT http://localhost:8081/api/v1/urls/google \
  -H "Content-Type: application/json" \
  -d '{
    "original_url": "https://www.google.com/search"
  }'
```

### 6. 🗑️ Delete URL

**REST API:**
```bash
# DELETE http://localhost:8081/api/v1/urls/{shortCode}
curl -X DELETE http://localhost:8081/api/v1/urls/google
```

### 7. 📊 Get Analytics

**REST API:**
```bash
# GET http://localhost:8081/api/v1/analytics/{shortCode}
curl http://localhost:8081/api/v1/analytics/google
```

**Response:**
```json
{
  "total_clicks": 15,
  "unique_clicks": 8,
  "clicks_by_date": [
    {
      "date": "2025-11-09",
      "clicks": 15
    }
  ],
  "top_referrers": [
    {
      "referrer": "direct",
      "clicks": 10
    }
  ]
}
```

### 8. 🏥 Health Check

**REST API:**
```bash
# GET http://localhost:8081/api/v1/health
curl http://localhost:8081/api/v1/health
```

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2025-11-09T12:12:38.687025127Z",
  "version": "1.0.0",
  "dependencies": {
    "database": "healthy",
    "cache": "healthy"
  }
}
```

---

## 🔌 **gRPC API Examples**

### 1. 🔗 Create Shortened URL

```bash
# With custom alias
grpcurl -plaintext -d '{
  "original_url": "https://www.google.com",
  "custom_alias": "google"
}' localhost:8080 url_shortener.v1.URLShortenerService/ShortenURL

# Without custom alias (auto-generated)
grpcurl -plaintext -d '{
  "original_url": "https://www.github.com"
}' localhost:8080 url_shortener.v1.URLShortenerService/ShortenURL

# With expiration (24 hours)
grpcurl -plaintext -d '{
  "original_url": "https://www.example.com",
  "expires_in": "86400s"
}' localhost:8080 url_shortener.v1.URLShortenerService/ShortenURL
```

**Response:**
```json
{
  "url": {
    "id": "4e632693-31fc-4d85-9c71-b176e57aa690",
    "shortCode": "google",
    "originalUrl": "https://www.google.com",
    "createdAt": "2025-11-09T12:12:28.480021970Z",
    "updatedAt": "2025-11-09T12:12:28.480022137Z",
    "customAlias": "google",
    "isActive": true
  },
  "shortUrl": "http://localhost:8080/google"
}
```

## 2. 🔍 Get Original URL (for redirection)

```bash
grpcurl -plaintext -d '{
  "short_code": "google"
}' localhost:8080 url_shortener.v1.URLShortenerService/GetOriginalURL
```

**Response:**
```json
{
  "originalUrl": "https://www.google.com",
  "isActive": true
}
```

## 3. ℹ️ Get URL Information

```bash
grpcurl -plaintext -d '{
  "short_code": "google"
}' localhost:8080 url_shortener.v1.URLShortenerService/GetURLInfo
```

**Response:**
```json
{
  "url": {
    "id": "4e632693-31fc-4d85-9c71-b176e57aa690",
    "shortCode": "google",
    "originalUrl": "https://www.google.com",
    "createdAt": "2025-11-09T12:12:28.480021970Z",
    "updatedAt": "2025-11-09T12:12:28.480022137Z",
    "clickCount": "5",
    "lastAccessedAt": "2025-11-09T12:15:30.123456789Z",
    "customAlias": "google",
    "isActive": true
  }
}
```

## 4. 📋 List URLs (with pagination)

```bash
# First page
grpcurl -plaintext -d '{
  "page_size": 10,
  "page_token": ""
}' localhost:8080 url_shortener.v1.URLShortenerService/ListURLs

# Next page (use next_page_token from previous response)
grpcurl -plaintext -d '{
  "page_size": 10,
  "page_token": "eyJvZmZzZXQiOjEwfQ=="
}' localhost:8080 url_shortener.v1.URLShortenerService/ListURLs
```

## 5. ✏️ Update URL

```bash
grpcurl -plaintext -d '{
  "short_code": "google",
  "original_url": "https://www.google.com/search"
}' localhost:8080 url_shortener.v1.URLShortenerService/UpdateURL
```

## 6. 🗑️ Delete URL

```bash
grpcurl -plaintext -d '{
  "short_code": "google"
}' localhost:8080 url_shortener.v1.URLShortenerService/DeleteURL
```

## 7. 📊 Get Analytics

```bash
grpcurl -plaintext -d '{
  "short_code": "google"
}' localhost:8080 url_shortener.v1.URLShortenerService/GetAnalytics
```

**Response:**
```json
{
  "totalClicks": "15",
  "uniqueClicks": "8",
  "clicksByDate": [
    {
      "date": "2025-11-09",
      "clicks": "15"
    }
  ],
  "topReferrers": [
    {
      "referrer": "direct",
      "clicks": "10"
    }
  ]
}
```

## 8. 🏥 Health Check

```bash
grpcurl -plaintext -d '{}' localhost:8080 url_shortener.v1.URLShortenerService/GetHealthCheck
```

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2025-11-09T12:12:38.687025127Z",
  "version": "1.0.0",
  "dependencies": {
    "cache": "healthy",
    "database": "healthy"
  }
}
```

## 🔍 Discover Available Services

```bash
# List all services
grpcurl -plaintext localhost:8080 list

# Describe a specific service
grpcurl -plaintext localhost:8080 describe url_shortener.v1.URLShortenerService

# Describe a specific method
grpcurl -plaintext localhost:8080 describe url_shortener.v1.URLShortenerService.ShortenURL
```

## 🚨 Error Handling Examples

```bash
# Invalid URL
grpcurl -plaintext -d '{
  "original_url": "not-a-valid-url"
}' localhost:8080 url_shortener.v1.URLShortenerService/ShortenURL

# Non-existent short code
grpcurl -plaintext -d '{
  "short_code": "nonexistent"
}' localhost:8080 url_shortener.v1.URLShortenerService/GetOriginalURL

# Duplicate custom alias
grpcurl -plaintext -d '{
  "original_url": "https://www.example.com",
  "custom_alias": "google"
}' localhost:8080 url_shortener.v1.URLShortenerService/ShortenURL
```
