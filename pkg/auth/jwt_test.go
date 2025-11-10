package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTManager_GenerateToken(t *testing.T) {
	secretKey := "test-secret-key-for-jwt-testing"
	duration := time.Hour
	issuer := "test-issuer"
	
	manager := NewJWTManager(secretKey, duration, issuer)

	tests := []struct {
		name     string
		userID   string
		username string
		email    string
		roles    []string
		wantErr  bool
	}{
		{
			name:     "valid admin user",
			userID:   "admin-123",
			username: "admin",
			email:    "admin@example.com",
			roles:    []string{"admin"},
			wantErr:  false,
		},
		{
			name:     "valid regular user",
			userID:   "user-456",
			username: "testuser",
			email:    "user@example.com",
			roles:    []string{"user"},
			wantErr:  false,
		},
		{
			name:     "user with multiple roles",
			userID:   "user-789",
			username: "poweruser",
			email:    "power@example.com",
			roles:    []string{"user", "moderator"},
			wantErr:  false,
		},
		{
			name:     "empty user ID",
			userID:   "",
			username: "testuser",
			email:    "user@example.com",
			roles:    []string{"user"},
			wantErr:  false, // Should still generate token
		},
		{
			name:     "empty roles",
			userID:   "user-000",
			username: "noroles",
			email:    "noroles@example.com",
			roles:    []string{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := manager.GenerateToken(tt.userID, tt.username, tt.email, tt.roles)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, token)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, token)
				
				// Verify token structure
				parts, err := jwt.NewParser().Parse(token, func(token *jwt.Token) (interface{}, error) {
					return []byte(secretKey), nil
				})
				assert.NoError(t, err)
				assert.NotNil(t, parts)
			}
		})
	}
}

func TestJWTManager_ValidateToken(t *testing.T) {
	secretKey := "test-secret-key-for-jwt-testing"
	duration := time.Hour
	issuer := "test-issuer"
	
	manager := NewJWTManager(secretKey, duration, issuer)

	// Generate a valid token
	validToken, err := manager.GenerateToken("user-123", "testuser", "test@example.com", []string{"user"})
	require.NoError(t, err)

	// Generate an expired token
	expiredManager := NewJWTManager(secretKey, -time.Hour, issuer) // Negative duration = expired
	expiredToken, err := expiredManager.GenerateToken("user-456", "expired", "expired@example.com", []string{"user"})
	require.NoError(t, err)

	tests := []struct {
		name      string
		token     string
		wantErr   bool
		wantClaims bool
	}{
		{
			name:       "valid token",
			token:      validToken,
			wantErr:    false,
			wantClaims: true,
		},
		{
			name:       "expired token",
			token:      expiredToken,
			wantErr:    true,
			wantClaims: false,
		},
		{
			name:       "invalid token format",
			token:      "invalid.token.format",
			wantErr:    true,
			wantClaims: false,
		},
		{
			name:       "empty token",
			token:      "",
			wantErr:    true,
			wantClaims: false,
		},
		{
			name:       "malformed token",
			token:      "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.malformed",
			wantErr:    true,
			wantClaims: false,
		},
		{
			name:       "token with wrong signature",
			token:      "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
			wantErr:    true,
			wantClaims: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := manager.ValidateToken(tt.token)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, claims)
			} else {
				assert.NoError(t, err)
				if tt.wantClaims {
					assert.NotNil(t, claims)
					assert.Equal(t, "user-123", claims.UserID)
					assert.Equal(t, "testuser", claims.Username)
					assert.Equal(t, "test@example.com", claims.Email)
					assert.Contains(t, claims.Roles, "user")
				}
			}
		})
	}
}

func TestJWTManager_RefreshToken(t *testing.T) {
	secretKey := "test-secret-key-for-jwt-testing"
	duration := time.Hour
	issuer := "test-issuer"
	
	manager := NewJWTManager(secretKey, duration, issuer)

	// Generate a token that's close to expiration
	shortDuration := 30 * time.Minute // Close to 1 hour limit
	shortManager := NewJWTManager(secretKey, shortDuration, issuer)
	nearExpiryToken, err := shortManager.GenerateToken("user-123", "testuser", "test@example.com", []string{"user"})
	require.NoError(t, err)

	// Generate a fresh token (not close to expiration) - create manager with longer duration
	freshManager := NewJWTManager(secretKey, 24*time.Hour, "test-issuer")
	freshToken, err := freshManager.GenerateToken("user-456", "freshuser", "fresh@example.com", []string{"user"})
	require.NoError(t, err)

	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name:    "token close to expiration",
			token:   nearExpiryToken,
			wantErr: false,
		},
		{
			name:    "fresh token (not close to expiration)",
			token:   freshToken,
			wantErr: true, // Should fail because token is not close to expiration
		},
		{
			name:    "invalid token",
			token:   "invalid.token",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newToken, err := manager.RefreshToken(tt.token)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, newToken)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, newToken)
				assert.NotEqual(t, tt.token, newToken) // New token should be different
			}
		})
	}
}

func TestClaims_HasRole(t *testing.T) {
	claims := &Claims{
		Roles: []string{"user", "admin", "moderator"},
	}

	tests := []struct {
		name string
		role string
		want bool
	}{
		{"has user role", "user", true},
		{"has admin role", "admin", true},
		{"has moderator role", "moderator", true},
		{"does not have super_admin role", "super_admin", false},
		{"empty role", "", false},
		{"case sensitive", "User", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, claims.HasRole(tt.role))
		})
	}
}

func TestClaims_IsAdmin(t *testing.T) {
	tests := []struct {
		name  string
		roles []string
		want  bool
	}{
		{"admin role", []string{"admin"}, true},
		{"admin with other roles", []string{"user", "admin", "moderator"}, true},
		{"super_admin role", []string{"super_admin"}, false}, // IsAdmin checks specifically for "admin"
		{"user role only", []string{"user"}, false},
		{"no roles", []string{}, false},
		{"empty roles", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims := &Claims{Roles: tt.roles}
			assert.Equal(t, tt.want, claims.IsAdmin())
		})
	}
}

func TestClaims_IsSuperAdmin(t *testing.T) {
	tests := []struct {
		name  string
		roles []string
		want  bool
	}{
		{"super_admin role", []string{"super_admin"}, true},
		{"super_admin with other roles", []string{"user", "super_admin", "admin"}, true},
		{"admin role only", []string{"admin"}, false},
		{"user role only", []string{"user"}, false},
		{"no roles", []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims := &Claims{Roles: tt.roles}
			assert.Equal(t, tt.want, claims.IsSuperAdmin())
		})
	}
}

func TestGenerateSecretKey(t *testing.T) {
	// Test multiple generations to ensure randomness
	keys := make(map[string]bool)
	
	for i := 0; i < 10; i++ {
		key, err := GenerateSecretKey()
		assert.NoError(t, err)
		assert.NotEmpty(t, key)
		assert.Len(t, key, 64) // 32 bytes * 2 (hex encoding) = 64 characters
		
		// Ensure each key is unique
		assert.False(t, keys[key], "Generated duplicate key: %s", key)
		keys[key] = true
	}
}

func TestJWTManager_TokenExpiration(t *testing.T) {
	secretKey := "test-secret-key"
	shortDuration := 500 * time.Millisecond
	issuer := "test-issuer"
	
	manager := NewJWTManager(secretKey, shortDuration, issuer)
	
	// Generate token
	token, err := manager.GenerateToken("user-123", "testuser", "test@example.com", []string{"user"})
	require.NoError(t, err)
	
	// Token should be valid immediately
	claims, err := manager.ValidateToken(token)
	require.NoError(t, err)
	require.NotNil(t, claims)
	
	// Wait for token to expire
	time.Sleep(600 * time.Millisecond)
	
	// Token should now be expired
	claims, err = manager.ValidateToken(token)
	require.Error(t, err)
	assert.Nil(t, claims)
}
