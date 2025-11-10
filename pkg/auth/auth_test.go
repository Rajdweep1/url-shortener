package auth

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAuthManager(t *testing.T) {
	tests := []struct {
		name    string
		config  AuthConfig
		wantErr bool
	}{
		{
			name: "valid config with both JWT and API key enabled",
			config: AuthConfig{
				JWTSecret:    "test-secret-key",
				JWTDuration:  time.Hour,
				JWTIssuer:    "test-issuer",
				EnableJWT:    true,
				EnableAPIKey: true,
			},
			wantErr: false,
		},
		{
			name: "JWT enabled without secret",
			config: AuthConfig{
				JWTSecret:    "",
				EnableJWT:    true,
				EnableAPIKey: false,
			},
			wantErr: true,
		},
		{
			name: "only API key enabled",
			config: AuthConfig{
				EnableJWT:    false,
				EnableAPIKey: true,
			},
			wantErr: false,
		},
		{
			name: "both disabled",
			config: AuthConfig{
				EnableJWT:    false,
				EnableAPIKey: false,
			},
			wantErr: false,
		},
		{
			name: "with admin API key",
			config: AuthConfig{
				JWTSecret:    "test-secret",
				EnableJWT:    true,
				EnableAPIKey: true,
				AdminAPIKey:  "admin-key-123",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewAuthManager(tt.config)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, manager)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, manager)
			}
		})
	}
}

func TestAuthManager_AuthenticateToken(t *testing.T) {
	config := AuthConfig{
		JWTSecret:    "test-secret-key",
		JWTDuration:  time.Hour,
		JWTIssuer:    "test-issuer",
		EnableJWT:    true,
		EnableAPIKey: true,
		AdminAPIKey:  "admin-key-123",
	}
	
	manager, err := NewAuthManager(config)
	require.NoError(t, err)
	require.NotNil(t, manager)

	// Generate a valid JWT token
	jwtToken, err := manager.GenerateJWT("user-123", "testuser", "test@example.com", []string{"user"})
	require.NoError(t, err)

	// Generate a valid API key
	apiKey, _, err := manager.GenerateAPIKey("Test Key", "user-456", []string{APIKeyPermissions.ReadURLs}, nil)
	require.NoError(t, err)

	tests := []struct {
		name      string
		token     string
		wantErr   bool
		wantType  string
		wantAdmin bool
	}{
		{
			name:      "valid JWT token",
			token:     jwtToken,
			wantErr:   false,
			wantType:  "jwt",
			wantAdmin: false,
		},
		{
			name:      "valid API key",
			token:     apiKey,
			wantErr:   false,
			wantType:  "api_key",
			wantAdmin: false,
		},
		{
			name:      "admin API key",
			token:     config.AdminAPIKey,
			wantErr:   false,
			wantType:  "api_key",
			wantAdmin: true,
		},
		{
			name:      "invalid token",
			token:     "invalid-token",
			wantErr:   true,
			wantType:  "",
			wantAdmin: false,
		},
		{
			name:      "empty token",
			token:     "",
			wantErr:   true,
			wantType:  "",
			wantAdmin: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authCtx, err := manager.AuthenticateToken(tt.token)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, authCtx)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, authCtx)
				assert.Equal(t, tt.wantType, authCtx.AuthType)
				assert.Equal(t, tt.wantAdmin, authCtx.IsAdmin)
			}
		})
	}
}

func TestAuthContext_RequirePermission(t *testing.T) {
	tests := []struct {
		name       string
		authCtx    *AuthContext
		permission string
		wantErr    bool
	}{
		{
			name: "admin has all permissions",
			authCtx: &AuthContext{
				IsAdmin: true,
			},
			permission: APIKeyPermissions.DeleteURLs,
			wantErr:    false,
		},
		{
			name: "user has required permission",
			authCtx: &AuthContext{
				IsAdmin:     false,
				Permissions: []string{APIKeyPermissions.ReadURLs, APIKeyPermissions.WriteURLs},
			},
			permission: APIKeyPermissions.ReadURLs,
			wantErr:    false,
		},
		{
			name: "user lacks required permission",
			authCtx: &AuthContext{
				IsAdmin:     false,
				Permissions: []string{APIKeyPermissions.ReadURLs},
			},
			permission: APIKeyPermissions.WriteURLs,
			wantErr:    true,
		},
		{
			name: "user with no permissions",
			authCtx: &AuthContext{
				IsAdmin:     false,
				Permissions: []string{},
			},
			permission: APIKeyPermissions.ReadURLs,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.authCtx.RequirePermission(tt.permission)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAuthContext_RequireRole(t *testing.T) {
	tests := []struct {
		name    string
		authCtx *AuthContext
		role    string
		wantErr bool
	}{
		{
			name: "admin has most roles",
			authCtx: &AuthContext{
				IsAdmin: true,
				Roles:   []string{"admin"},
			},
			role:    "moderator",
			wantErr: false,
		},
		{
			name: "admin cannot access super_admin",
			authCtx: &AuthContext{
				IsAdmin: true,
				Roles:   []string{"admin"},
			},
			role:    "super_admin",
			wantErr: true,
		},
		{
			name: "user has required role",
			authCtx: &AuthContext{
				IsAdmin: false,
				Roles:   []string{"user", "moderator"},
			},
			role:    "moderator",
			wantErr: false,
		},
		{
			name: "user lacks required role",
			authCtx: &AuthContext{
				IsAdmin: false,
				Roles:   []string{"user"},
			},
			role:    "admin",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.authCtx.RequireRole(tt.role)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestWithAuthContext_FromContext(t *testing.T) {
	authCtx := &AuthContext{
		UserID:   "user-123",
		Username: "testuser",
		IsAdmin:  false,
	}

	// Test adding auth context to context
	ctx := context.Background()
	ctxWithAuth := WithAuthContext(ctx, authCtx)

	// Test retrieving auth context from context
	retrievedAuthCtx, ok := FromContext(ctxWithAuth)
	assert.True(t, ok)
	assert.NotNil(t, retrievedAuthCtx)
	assert.Equal(t, authCtx.UserID, retrievedAuthCtx.UserID)
	assert.Equal(t, authCtx.Username, retrievedAuthCtx.Username)
	assert.Equal(t, authCtx.IsAdmin, retrievedAuthCtx.IsAdmin)

	// Test retrieving from context without auth
	emptyCtx := context.Background()
	retrievedAuthCtx, ok = FromContext(emptyCtx)
	assert.False(t, ok)
	assert.Nil(t, retrievedAuthCtx)
}

func TestAuthManager_GenerateJWT_Disabled(t *testing.T) {
	config := AuthConfig{
		EnableJWT:    false,
		EnableAPIKey: true,
	}
	
	manager, err := NewAuthManager(config)
	require.NoError(t, err)

	token, err := manager.GenerateJWT("user-123", "testuser", "test@example.com", []string{"user"})
	assert.Error(t, err)
	assert.Empty(t, token)
	assert.Contains(t, err.Error(), "JWT authentication is not enabled")
}

func TestAuthManager_GenerateAPIKey_Disabled(t *testing.T) {
	config := AuthConfig{
		JWTSecret:    "test-secret",
		EnableJWT:    true,
		EnableAPIKey: false,
	}
	
	manager, err := NewAuthManager(config)
	require.NoError(t, err)

	apiKey, keyInfo, err := manager.GenerateAPIKey("Test Key", "user-123", []string{APIKeyPermissions.ReadURLs}, nil)
	assert.Error(t, err)
	assert.Empty(t, apiKey)
	assert.Nil(t, keyInfo)
	assert.Contains(t, err.Error(), "API key authentication is not enabled")
}

func TestAuthManager_AuthenticateToken_NoMethodsEnabled(t *testing.T) {
	config := AuthConfig{
		EnableJWT:    false,
		EnableAPIKey: false,
	}
	
	manager, err := NewAuthManager(config)
	require.NoError(t, err)

	authCtx, err := manager.AuthenticateToken("any-token")
	assert.Error(t, err)
	assert.Nil(t, authCtx)
	assert.Contains(t, err.Error(), "no authentication methods enabled")
}
