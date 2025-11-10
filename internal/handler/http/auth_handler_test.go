package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rajweepmondal/url-shortener/pkg/auth"
)

func setupTestAuthHandler(t *testing.T) (*AuthHandler, *auth.AuthManager) {
	config := auth.AuthConfig{
		JWTSecret:    "test-secret-key-for-handler-testing",
		JWTDuration:  time.Hour,
		JWTIssuer:    "test-issuer",
		EnableJWT:    true,
		EnableAPIKey: true,
		AdminAPIKey:  "admin-key-123",
	}
	
	authManager, err := auth.NewAuthManager(config)
	require.NoError(t, err)
	
	logger := zap.NewNop()
	handler := NewAuthHandler(authManager, logger)
	
	return handler, authManager
}

func TestAuthHandler_Login(t *testing.T) {
	handler, _ := setupTestAuthHandler(t)

	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
		expectToken    bool
	}{
		{
			name: "valid admin credentials",
			requestBody: LoginRequest{
				Username: "admin",
				Password: "admin",
			},
			expectedStatus: http.StatusOK,
			expectToken:    true,
		},
		{
			name: "invalid username",
			requestBody: LoginRequest{
				Username: "wronguser",
				Password: "admin",
			},
			expectedStatus: http.StatusUnauthorized,
			expectToken:    false,
		},
		{
			name: "invalid password",
			requestBody: LoginRequest{
				Username: "admin",
				Password: "wrongpassword",
			},
			expectedStatus: http.StatusUnauthorized,
			expectToken:    false,
		},
		{
			name: "empty username",
			requestBody: LoginRequest{
				Username: "",
				Password: "admin",
			},
			expectedStatus: http.StatusUnauthorized,
			expectToken:    false,
		},
		{
			name: "empty password",
			requestBody: LoginRequest{
				Username: "admin",
				Password: "",
			},
			expectedStatus: http.StatusUnauthorized,
			expectToken:    false,
		},
		{
			name:           "invalid JSON",
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
			expectToken:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body []byte
			var err error
			
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				require.NoError(t, err)
			}
			
			req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			
			rr := httptest.NewRecorder()
			handler.Login(rr, req)
			
			assert.Equal(t, tt.expectedStatus, rr.Code)
			
			if tt.expectToken {
				var response LoginResponse
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.NotEmpty(t, response.Token)
				assert.Equal(t, "admin", response.User.Username)
				assert.Contains(t, response.User.Roles, "admin")
			}
		})
	}
}

func TestAuthHandler_CreateAPIKey(t *testing.T) {
	handler, authManager := setupTestAuthHandler(t)

	// Generate admin JWT token
	adminToken, err := authManager.GenerateJWT("admin-123", "admin", "admin@example.com", []string{"admin"})
	require.NoError(t, err)
	
	// Generate user JWT token
	userToken, err := authManager.GenerateJWT("user-456", "user", "user@example.com", []string{"user"})
	require.NoError(t, err)

	tests := []struct {
		name           string
		authHeader     string
		requestBody    interface{}
		expectedStatus int
		expectAPIKey   bool
	}{
		{
			name:       "admin creates API key",
			authHeader: "Bearer " + adminToken,
			requestBody: CreateAPIKeyRequest{
				Name:        "Test API Key",
				Permissions: []string{auth.APIKeyPermissions.ReadURLs, auth.APIKeyPermissions.WriteURLs},
			},
			expectedStatus: http.StatusCreated,
			expectAPIKey:   true,
		},
		{
			name:       "admin creates API key with expiration",
			authHeader: "Bearer " + adminToken,
			requestBody: CreateAPIKeyRequest{
				Name:        "Expiring Key",
				Permissions: []string{auth.APIKeyPermissions.ReadURLs},
				ExpiresAt:   timePtr(time.Now().Add(24 * time.Hour)),
			},
			expectedStatus: http.StatusCreated,
			expectAPIKey:   true,
		},
		{
			name:       "admin creates API key with default permissions",
			authHeader: "Bearer " + adminToken,
			requestBody: CreateAPIKeyRequest{
				Name: "Default Perms Key",
			},
			expectedStatus: http.StatusCreated,
			expectAPIKey:   true,
		},
		{
			name:       "user tries to create API key",
			authHeader: "Bearer " + userToken,
			requestBody: CreateAPIKeyRequest{
				Name:        "User Key",
				Permissions: []string{auth.APIKeyPermissions.ReadURLs},
			},
			expectedStatus: http.StatusForbidden,
			expectAPIKey:   false,
		},
		{
			name:       "no authentication",
			authHeader: "",
			requestBody: CreateAPIKeyRequest{
				Name:        "No Auth Key",
				Permissions: []string{auth.APIKeyPermissions.ReadURLs},
			},
			expectedStatus: http.StatusUnauthorized,
			expectAPIKey:   false,
		},
		{
			name:           "invalid JSON",
			authHeader:     "Bearer " + adminToken,
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
			expectAPIKey:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body []byte
			var err error
			
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				require.NoError(t, err)
			}
			
			req := httptest.NewRequest("POST", "/api/v1/auth/api-keys", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			
			// Add auth context for authenticated requests
			if tt.authHeader != "" && tt.expectedStatus != http.StatusUnauthorized {
				authCtx, err := authManager.AuthenticateToken(strings.TrimPrefix(tt.authHeader, "Bearer "))
				if err == nil {
					ctx := auth.WithAuthContext(req.Context(), authCtx)
					req = req.WithContext(ctx)
				}
			}
			
			rr := httptest.NewRecorder()
			handler.CreateAPIKey(rr, req)
			
			assert.Equal(t, tt.expectedStatus, rr.Code)
			
			if tt.expectAPIKey {
				var response CreateAPIKeyResponse
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.NotEmpty(t, response.APIKey)
				assert.Contains(t, response.APIKey, "usk_")
				assert.NotNil(t, response.KeyInfo)
			}
		})
	}
}

func TestAuthHandler_ValidateToken(t *testing.T) {
	handler, authManager := setupTestAuthHandler(t)

	// Generate valid tokens
	jwtToken, err := authManager.GenerateJWT("user-123", "testuser", "test@example.com", []string{"user"})
	require.NoError(t, err)
	
	apiKey, _, err := authManager.GenerateAPIKey("Test Key", "user-456", []string{auth.APIKeyPermissions.ReadURLs}, nil)
	require.NoError(t, err)

	tests := []struct {
		name           string
		authHeader     string
		queryParam     string
		expectedStatus int
		expectValid    bool
	}{
		{
			name:           "valid JWT token in header",
			authHeader:     "Bearer " + jwtToken,
			expectedStatus: http.StatusOK,
			expectValid:    true,
		},
		{
			name:           "valid API key in header",
			authHeader:     "ApiKey " + apiKey,
			expectedStatus: http.StatusOK,
			expectValid:    true,
		},
		{
			name:           "valid token in query param",
			queryParam:     jwtToken,
			expectedStatus: http.StatusOK,
			expectValid:    true,
		},
		{
			name:           "invalid token",
			authHeader:     "Bearer invalid-token",
			expectedStatus: http.StatusUnauthorized,
			expectValid:    false,
		},
		{
			name:           "no token",
			expectedStatus: http.StatusBadRequest,
			expectValid:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/v1/auth/validate"
			if tt.queryParam != "" {
				url += "?token=" + tt.queryParam
			}
			
			req := httptest.NewRequest("POST", url, nil)
			
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			
			rr := httptest.NewRecorder()
			handler.ValidateToken(rr, req)
			
			assert.Equal(t, tt.expectedStatus, rr.Code)
			
			if tt.expectValid {
				var response map[string]interface{}
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.True(t, response["valid"].(bool))
				assert.NotEmpty(t, response["user_id"])
			}
		})
	}
}

func TestAuthHandler_GetProfile(t *testing.T) {
	handler, authManager := setupTestAuthHandler(t)

	// Generate test token (not used in this test but validates auth manager works)
	_, err := authManager.GenerateJWT("user-123", "testuser", "test@example.com", []string{"user"})
	require.NoError(t, err)

	tests := []struct {
		name           string
		hasAuthContext bool
		authContext    *auth.AuthContext
		expectedStatus int
	}{
		{
			name:           "valid auth context",
			hasAuthContext: true,
			authContext: &auth.AuthContext{
				UserID:   "user-123",
				Username: "testuser",
				Email:    "test@example.com",
				Roles:    []string{"user"},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "no auth context",
			hasAuthContext: false,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/auth/profile", nil)
			
			if tt.hasAuthContext {
				ctx := auth.WithAuthContext(req.Context(), tt.authContext)
				req = req.WithContext(ctx)
			}
			
			rr := httptest.NewRecorder()
			handler.GetProfile(rr, req)
			
			assert.Equal(t, tt.expectedStatus, rr.Code)
			
			if tt.expectedStatus == http.StatusOK {
				var profile UserInfo
				err := json.Unmarshal(rr.Body.Bytes(), &profile)
				assert.NoError(t, err)
				assert.Equal(t, tt.authContext.UserID, profile.ID)
				assert.Equal(t, tt.authContext.Username, profile.Username)
				assert.Equal(t, tt.authContext.Email, profile.Email)
				assert.Equal(t, tt.authContext.Roles, profile.Roles)
			}
		})
	}
}

func TestAuthHandler_extractToken(t *testing.T) {
	handler, _ := setupTestAuthHandler(t)

	tests := []struct {
		name     string
		headers  map[string]string
		query    map[string]string
		expected string
	}{
		{
			name: "Bearer token",
			headers: map[string]string{
				"Authorization": "Bearer test-token-123",
			},
			expected: "test-token-123",
		},
		{
			name: "ApiKey token",
			headers: map[string]string{
				"Authorization": "ApiKey test-api-key-456",
			},
			expected: "test-api-key-456",
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
				"token": "query-token-123",
			},
			expected: "query-token-123",
		},
		{
			name:     "No token",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}
			
			q := req.URL.Query()
			for key, value := range tt.query {
				q.Set(key, value)
			}
			req.URL.RawQuery = q.Encode()
			
			token := handler.extractToken(req)
			assert.Equal(t, tt.expected, token)
		})
	}
}

// Helper function to create time pointer
func timePtr(t time.Time) *time.Time {
	return &t
}
