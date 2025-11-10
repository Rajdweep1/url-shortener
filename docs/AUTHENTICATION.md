# Authentication System

The URL Shortener service implements a comprehensive authentication system supporting both JWT tokens and API keys for secure access to protected endpoints.

## Overview

The authentication system provides:
- **JWT Token Authentication** - For user sessions and web applications
- **API Key Authentication** - For service-to-service communication
- **Role-based Access Control** - Admin and user roles with different permissions
- **Flexible Configuration** - Enable/disable authentication methods as needed

## Configuration

Authentication is configured via environment variables:

```bash
# Authentication Configuration
AUTH_JWT_SECRET=your-super-secret-jwt-key-change-this-in-production
AUTH_JWT_DURATION=24h
AUTH_JWT_ISSUER=url-shortener
AUTH_ADMIN_API_KEY=usk_admin_key_change_this_in_production
AUTH_ENABLE_JWT=true
AUTH_ENABLE_API_KEY=true
AUTH_REQUIRE_AUTH=false  # Set to true in production
```

## JWT Authentication

### Login
Obtain a JWT token by authenticating with username/password:

```bash
curl -X POST http://localhost:8081/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "admin"}'
```

Response:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_at": "2024-01-02T00:00:00Z",
  "user": {
    "id": "admin",
    "username": "admin",
    "email": "admin@example.com",
    "roles": ["admin"]
  }
}
```

### Using JWT Token
Include the JWT token in the Authorization header:

```bash
curl -X GET http://localhost:8081/api/v1/auth/profile \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

## API Key Authentication

### Creating API Keys
Create an API key (requires admin JWT token):

```bash
curl -X POST http://localhost:8081/api/v1/auth/api-keys \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "name": "My API Key",
    "permissions": ["urls:read", "urls:write"],
    "expires_at": "2025-01-01T00:00:00Z"
  }'
```

Response:
```json
{
  "api_key": "usk_abcd1234...",
  "key_info": {
    "id": "key123",
    "name": "My API Key",
    "permissions": ["urls:read", "urls:write"],
    "created_at": "2024-01-01T00:00:00Z",
    "expires_at": "2025-01-01T00:00:00Z",
    "is_active": true
  }
}
```

### Using API Keys
Include the API key in headers:

```bash
# Using X-API-Key header
curl -X GET http://localhost:8081/api/v1/urls \
  -H "X-API-Key: usk_abcd1234..."

# Using Authorization header
curl -X GET http://localhost:8081/api/v1/urls \
  -H "Authorization: ApiKey usk_abcd1234..."
```

## Permissions

The system supports granular permissions:

- `urls:read` - Read URL information and list URLs
- `urls:write` - Create and update URLs
- `urls:delete` - Delete URLs
- `analytics:read` - Access analytics data
- `admin:access` - Full admin access (includes all permissions)

## Endpoints

### Public Endpoints (No Authentication Required)
- `GET /health` - Health check
- `GET /api/v1/health` - Detailed health check
- `GET /{shortCode}` - URL redirection
- `POST /api/v1/auth/login` - User login
- `POST /api/v1/auth/validate` - Token validation

### Protected Endpoints (Authentication Required)
- `GET /api/v1/auth/profile` - Get user profile
- `POST /api/v1/auth/api-keys` - Create API key (admin only)
- `POST /api/v1/urls` - Create short URL
- `GET /api/v1/urls` - List URLs
- `GET /api/v1/urls/{shortCode}` - Get URL info
- `PUT /api/v1/urls/{shortCode}` - Update URL
- `DELETE /api/v1/urls/{shortCode}` - Delete URL (admin only)
- `GET /api/v1/analytics/{shortCode}` - Get analytics (admin only)

## Testing Authentication

Use the provided test script to verify authentication:

```bash
./scripts/test-auth.sh
```

This script tests:
- Login with valid/invalid credentials
- JWT token validation
- API key creation and usage
- Protected endpoint access
- Unauthorized access handling

## Development vs Production

### Development Mode
Set `AUTH_REQUIRE_AUTH=false` to disable authentication for development:
- All endpoints are accessible without authentication
- Useful for testing and development

### Production Mode
Set `AUTH_REQUIRE_AUTH=true` to enforce authentication:
- Protected endpoints require valid JWT token or API key
- Admin endpoints require admin privileges
- Proper error handling for unauthorized access

## Security Best Practices

1. **Strong JWT Secret**: Use a cryptographically secure secret key (32+ characters)
2. **HTTPS Only**: Always use HTTPS in production
3. **Token Expiration**: Set appropriate JWT expiration times
4. **API Key Rotation**: Regularly rotate API keys
5. **Principle of Least Privilege**: Grant minimal required permissions
6. **Secure Storage**: Never log or expose tokens/keys in plain text

## Error Responses

Authentication errors return structured JSON responses:

```json
{
  "error": {
    "message": "Invalid or expired token",
    "code": 401
  },
  "timestamp": "2024-01-01T00:00:00Z"
}
```

Common error codes:
- `401 Unauthorized` - Missing or invalid authentication
- `403 Forbidden` - Valid authentication but insufficient permissions
- `400 Bad Request` - Invalid request format

## Integration Examples

### cURL Examples
```bash
# Login and get JWT
TOKEN=$(curl -s -X POST http://localhost:8081/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "admin"}' | jq -r '.token')

# Create URL with JWT
curl -X POST http://localhost:8081/api/v1/urls \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"original_url": "https://example.com"}'
```

### JavaScript Example
```javascript
// Login
const loginResponse = await fetch('/api/v1/auth/login', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ username: 'admin', password: 'admin' })
});
const { token } = await loginResponse.json();

// Use token for API calls
const urlResponse = await fetch('/api/v1/urls', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${token}`
  },
  body: JSON.stringify({ original_url: 'https://example.com' })
});
```

## Troubleshooting

### Common Issues

1. **"JWT secret is required"**
   - Set `AUTH_JWT_SECRET` environment variable

2. **"Invalid token"**
   - Check token format and expiration
   - Ensure correct Authorization header format

3. **"Admin access required"**
   - User lacks admin role for the requested operation
   - Use admin credentials or API key with admin permissions

4. **"Authentication required"**
   - Missing Authorization header
   - Check if `AUTH_REQUIRE_AUTH=true` in production
