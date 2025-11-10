package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rajweepmondal/url-shortener/pkg/auth"
)

func setupTestAuthManager(t *testing.T) *auth.AuthManager {
	config := auth.AuthConfig{
		JWTSecret:    "test-secret-key-for-middleware-testing",
		JWTDuration:  time.Hour,
		JWTIssuer:    "test-issuer",
		EnableJWT:    true,
		EnableAPIKey: true,
		AdminAPIKey:  "admin-key-123",
	}
	
	manager, err := auth.NewAuthManager(config)
	require.NoError(t, err)
	return manager
}

func TestAuthMiddleware_HTTPAuthMiddleware(t *testing.T) {
	authManager := setupTestAuthManager(t)
	logger := zap.NewNop()
	
	// Generate test tokens
	jwtToken, err := authManager.GenerateJWT("user-123", "testuser", "test@example.com", []string{"user"})
	require.NoError(t, err)
	
	adminJWTToken, err := authManager.GenerateJWT("admin-456", "admin", "admin@example.com", []string{"admin"})
	require.NoError(t, err)
	
	apiKey, _, err := authManager.GenerateAPIKey("Test Key", "user-789", []string{auth.APIKeyPermissions.ReadURLs}, nil)
	require.NoError(t, err)

	tests := []struct {
		name           string
		path           string
		method         string
		authHeader     string
		requireAuth    bool
		expectedStatus int
		expectAuthCtx  bool
	}{
		{
			name:           "public endpoint - health check",
			path:           "/api/v1/health",
			method:         "GET",
			authHeader:     "",
			requireAuth:    true,
			expectedStatus: http.StatusOK,
			expectAuthCtx:  false,
		},
		{
			name:           "public endpoint - redirect",
			path:           "/abc123",
			method:         "GET",
			authHeader:     "",
			requireAuth:    true,
			expectedStatus: http.StatusOK,
			expectAuthCtx:  false,
		},
		{
			name:           "protected endpoint with valid JWT",
			path:           "/api/v1/urls",
			method:         "POST",
			authHeader:     "Bearer " + jwtToken,
			requireAuth:    true,
			expectedStatus: http.StatusOK,
			expectAuthCtx:  true,
		},
		{
			name:           "protected endpoint with valid API key",
			path:           "/api/v1/urls",
			method:         "GET",
			authHeader:     "ApiKey " + apiKey,
			requireAuth:    true,
			expectedStatus: http.StatusOK,
			expectAuthCtx:  true,
		},
		{
			name:           "protected endpoint without auth",
			path:           "/api/v1/urls",
			method:         "POST",
			authHeader:     "",
			requireAuth:    true,
			expectedStatus: http.StatusUnauthorized,
			expectAuthCtx:  false,
		},
		{
			name:           "protected endpoint with invalid token",
			path:           "/api/v1/urls",
			method:         "POST",
			authHeader:     "Bearer invalid-token",
			requireAuth:    true,
			expectedStatus: http.StatusUnauthorized,
			expectAuthCtx:  false,
		},
		{
			name:           "admin endpoint with admin JWT",
			path:           "/api/v1/urls",
			method:         "DELETE",
			authHeader:     "Bearer " + adminJWTToken,
			requireAuth:    true,
			expectedStatus: http.StatusOK,
			expectAuthCtx:  true,
		},
		{
			name:           "admin endpoint with user JWT",
			path:           "/api/v1/urls",
			method:         "DELETE",
			authHeader:     "Bearer " + jwtToken,
			requireAuth:    true,
			expectedStatus: http.StatusForbidden,
			expectAuthCtx:  false,
		},
		{
			name:           "auth disabled - protected endpoint accessible",
			path:           "/api/v1/urls",
			method:         "POST",
			authHeader:     "",
			requireAuth:    false,
			expectedStatus: http.StatusOK,
			expectAuthCtx:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authMiddleware := NewAuthMiddleware(authManager, logger, tt.requireAuth)
			
			// Create a test handler that checks for auth context
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				authCtx, hasAuth := auth.FromContext(r.Context())
				
				if tt.expectAuthCtx {
					assert.True(t, hasAuth, "Expected auth context but didn't find it")
					assert.NotNil(t, authCtx, "Auth context should not be nil")
				} else if tt.expectedStatus == http.StatusOK {
					// For public endpoints, we don't expect auth context
					assert.False(t, hasAuth, "Didn't expect auth context for public endpoint")
				}
				
				w.WriteHeader(http.StatusOK)
			})
			
			// Wrap with auth middleware
			handler := authMiddleware.HTTPAuthMiddleware()(testHandler)
			
			// Create request
			req := httptest.NewRequest(tt.method, tt.path, nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			
			// Create response recorder
			rr := httptest.NewRecorder()
			
			// Execute request
			handler.ServeHTTP(rr, req)
			
			// Check status code
			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestAuthMiddleware_HTTPAuthMiddleware_HeaderFormats(t *testing.T) {
	authManager := setupTestAuthManager(t)
	logger := zap.NewNop()
	authMiddleware := NewAuthMiddleware(authManager, logger, true)
	
	// Generate test API key
	apiKey, _, err := authManager.GenerateAPIKey("Test Key", "user-123", []string{auth.APIKeyPermissions.ReadURLs}, nil)
	require.NoError(t, err)

	tests := []struct {
		name       string
		headers    map[string]string
		wantStatus int
	}{
		{
			name: "Authorization Bearer header",
			headers: map[string]string{
				"Authorization": "Bearer " + apiKey,
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "Authorization ApiKey header",
			headers: map[string]string{
				"Authorization": "ApiKey " + apiKey,
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "X-API-Key header",
			headers: map[string]string{
				"X-API-Key": apiKey,
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "Direct Authorization header",
			headers: map[string]string{
				"Authorization": apiKey,
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
			
			handler := authMiddleware.HTTPAuthMiddleware()(testHandler)
			
			req := httptest.NewRequest("POST", "/api/v1/urls", nil)
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}
			
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			
			assert.Equal(t, tt.wantStatus, rr.Code)
		})
	}
}

func TestAuthMiddleware_GRPCAuthInterceptor(t *testing.T) {
	authManager := setupTestAuthManager(t)
	logger := zap.NewNop()
	
	// Generate test tokens
	jwtToken, err := authManager.GenerateJWT("user-123", "testuser", "test@example.com", []string{"user"})
	require.NoError(t, err)
	
	adminJWTToken, err := authManager.GenerateJWT("admin-456", "admin", "admin@example.com", []string{"admin"})
	require.NoError(t, err)

	tests := []struct {
		name         string
		method       string
		metadata     map[string]string
		requireAuth  bool
		expectError  bool
		expectAdmin  bool
	}{
		{
			name:        "public endpoint - health check",
			method:      "/url_shortener.v1.URLShortenerService/GetHealthCheck",
			metadata:    map[string]string{},
			requireAuth: true,
			expectError: false,
		},
		{
			name:        "public endpoint - get original URL",
			method:      "/url_shortener.v1.URLShortenerService/GetOriginalURL",
			metadata:    map[string]string{},
			requireAuth: true,
			expectError: false,
		},
		{
			name:   "protected endpoint with valid JWT",
			method: "/url_shortener.v1.URLShortenerService/ShortenURL",
			metadata: map[string]string{
				"authorization": "Bearer " + jwtToken,
			},
			requireAuth: true,
			expectError: false,
		},
		{
			name:        "protected endpoint without auth",
			method:      "/url_shortener.v1.URLShortenerService/ShortenURL",
			metadata:    map[string]string{},
			requireAuth: true,
			expectError: true,
		},
		{
			name:   "admin endpoint with admin JWT",
			method: "/url_shortener.v1.URLShortenerService/DeleteURL",
			metadata: map[string]string{
				"authorization": "Bearer " + adminJWTToken,
			},
			requireAuth: true,
			expectError: false,
			expectAdmin: true,
		},
		{
			name:   "admin endpoint with user JWT",
			method: "/url_shortener.v1.URLShortenerService/DeleteURL",
			metadata: map[string]string{
				"authorization": "Bearer " + jwtToken,
			},
			requireAuth: true,
			expectError: true,
		},
		{
			name:        "auth disabled",
			method:      "/url_shortener.v1.URLShortenerService/ShortenURL",
			metadata:    map[string]string{},
			requireAuth: false,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authMiddleware := NewAuthMiddleware(authManager, logger, tt.requireAuth)
			
			// Create test handler
			testHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
				if tt.expectAdmin {
					authCtx, ok := auth.FromContext(ctx)
					assert.True(t, ok, "Expected auth context")
					assert.True(t, authCtx.IsAdmin, "Expected admin context")
				}
				return "success", nil
			}
			
			// Create gRPC info
			info := &grpc.UnaryServerInfo{
				FullMethod: tt.method,
			}
			
			// Create context with metadata
			ctx := context.Background()
			if len(tt.metadata) > 0 {
				md := metadata.New(tt.metadata)
				ctx = metadata.NewIncomingContext(ctx, md)
			}
			
			// Execute interceptor
			interceptor := authMiddleware.GRPCAuthInterceptor()
			result, err := interceptor(ctx, nil, info, testHandler)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "success", result)
			}
		})
	}
}

func TestAuthMiddleware_extractToken(t *testing.T) {
	authManager := setupTestAuthManager(t)
	logger := zap.NewNop()
	authMiddleware := NewAuthMiddleware(authManager, logger, true)

	tests := []struct {
		name     string
		headers  map[string]string
		query    map[string]string
		expected string
	}{
		{
			name: "Bearer token in Authorization header",
			headers: map[string]string{
				"Authorization": "Bearer test-token-123",
			},
			expected: "test-token-123",
		},
		{
			name: "ApiKey in Authorization header",
			headers: map[string]string{
				"Authorization": "ApiKey test-api-key-456",
			},
			expected: "test-api-key-456",
		},
		{
			name: "Direct token in Authorization header",
			headers: map[string]string{
				"Authorization": "direct-token-789",
			},
			expected: "direct-token-789",
		},
		{
			name: "X-API-Key header",
			headers: map[string]string{
				"X-API-Key": "x-api-key-token",
			},
			expected: "x-api-key-token",
		},
		{
			name: "Query parameter",
			query: map[string]string{
				"api_key": "query-token-123",
			},
			expected: "query-token-123",
		},
		{
			name:     "No token",
			headers:  map[string]string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			
			// Set headers
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}
			
			// Set query parameters
			q := req.URL.Query()
			for key, value := range tt.query {
				q.Set(key, value)
			}
			req.URL.RawQuery = q.Encode()
			
			token := authMiddleware.extractToken(req)
			assert.Equal(t, tt.expected, token)
		})
	}
}

func TestAuthMiddleware_isPublicEndpoint(t *testing.T) {
	authManager := setupTestAuthManager(t)
	logger := zap.NewNop()
	authMiddleware := NewAuthMiddleware(authManager, logger, true)

	tests := []struct {
		name     string
		path     string
		method   string
		expected bool
	}{
		{"health endpoint", "/health", "GET", true},
		{"api health endpoint", "/api/v1/health", "GET", true},
		{"root endpoint", "/", "GET", true},
		{"login endpoint", "/api/v1/auth/login", "POST", true},
		{"validate endpoint", "/api/v1/auth/validate", "POST", true},
		{"redirect endpoint", "/abc123", "GET", true},
		{"protected API endpoint", "/api/v1/urls", "POST", false},
		{"protected profile endpoint", "/api/v1/auth/profile", "GET", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			result := authMiddleware.isPublicEndpoint(req)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAuthMiddleware_isAdminEndpoint(t *testing.T) {
	authManager := setupTestAuthManager(t)
	logger := zap.NewNop()
	authMiddleware := NewAuthMiddleware(authManager, logger, true)

	tests := []struct {
		name     string
		path     string
		method   string
		expected bool
	}{
		{"GET URLs (read operation)", "/api/v1/urls", "GET", false},
		{"POST URLs (write operation)", "/api/v1/urls", "POST", false},
		{"PUT URLs (write operation)", "/api/v1/urls/abc123", "PUT", false},
		{"DELETE URLs (admin operation)", "/api/v1/urls/abc123", "DELETE", true},
		{"GET Analytics", "/api/v1/analytics/abc123", "GET", true},
		{"Non-admin endpoint", "/api/v1/auth/profile", "GET", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			result := authMiddleware.isAdminEndpoint(req)
			assert.Equal(t, tt.expected, result)
		})
	}
}
