package auth

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIKeyManager_GenerateAPIKey(t *testing.T) {
	manager := NewAPIKeyManager()

	tests := []struct {
		name        string
		keyName     string
		userID      string
		permissions []string
		expiresAt   *time.Time
		wantErr     bool
	}{
		{
			name:        "valid API key with permissions",
			keyName:     "Test Key",
			userID:      "user-123",
			permissions: []string{APIKeyPermissions.ReadURLs, APIKeyPermissions.WriteURLs},
			expiresAt:   nil,
			wantErr:     false,
		},
		{
			name:        "API key with expiration",
			keyName:     "Expiring Key",
			userID:      "user-456",
			permissions: []string{APIKeyPermissions.AdminAccess},
			expiresAt:   timePtr(time.Now().Add(24 * time.Hour)),
			wantErr:     false,
		},
		{
			name:        "API key with all permissions",
			keyName:     "Admin Key",
			userID:      "admin-789",
			permissions: []string{
				APIKeyPermissions.ReadURLs,
				APIKeyPermissions.WriteURLs,
				APIKeyPermissions.DeleteURLs,
				APIKeyPermissions.ReadAnalytics,
				APIKeyPermissions.AdminAccess,
			},
			expiresAt: nil,
			wantErr:   false,
		},
		{
			name:        "empty key name",
			keyName:     "",
			userID:      "user-000",
			permissions: []string{APIKeyPermissions.ReadURLs},
			expiresAt:   nil,
			wantErr:     false, // Should still work
		},
		{
			name:        "empty permissions",
			keyName:     "No Perms Key",
			userID:      "user-111",
			permissions: []string{},
			expiresAt:   nil,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiKey, keyInfo, err := manager.GenerateAPIKey(tt.keyName, tt.userID, tt.permissions, tt.expiresAt)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, apiKey)
				assert.Nil(t, keyInfo)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, apiKey)
				assert.NotNil(t, keyInfo)
				
				// Verify API key format
				assert.True(t, len(apiKey) > 10, "API key should be reasonably long")
				assert.Contains(t, apiKey, "usk_", "API key should have correct prefix")
				
				// Verify key info
				assert.Equal(t, tt.keyName, keyInfo.Name)
				assert.Equal(t, tt.userID, keyInfo.UserID)
				assert.Equal(t, tt.permissions, keyInfo.Permissions)
				assert.Equal(t, tt.expiresAt, keyInfo.ExpiresAt)
				assert.True(t, keyInfo.IsActive)
				assert.NotEmpty(t, keyInfo.ID)
				assert.NotEmpty(t, keyInfo.HashedKey)
				assert.False(t, keyInfo.CreatedAt.IsZero())
			}
		})
	}
}

func TestAPIKeyManager_ValidateAPIKey(t *testing.T) {
	manager := NewAPIKeyManager()
	
	// Generate a valid API key
	validKey, keyInfo, err := manager.GenerateAPIKey("Test Key", "user-123", []string{APIKeyPermissions.ReadURLs}, nil)
	require.NoError(t, err)
	require.NotNil(t, keyInfo)
	
	// Generate an expired API key
	expiredTime := time.Now().Add(-time.Hour)
	expiredKey, expiredKeyInfo, err := manager.GenerateAPIKey("Expired Key", "user-456", []string{APIKeyPermissions.ReadURLs}, &expiredTime)
	require.NoError(t, err)
	require.NotNil(t, expiredKeyInfo)
	
	// Generate an inactive API key
	inactiveKey, inactiveKeyInfo, err := manager.GenerateAPIKey("Inactive Key", "user-789", []string{APIKeyPermissions.ReadURLs}, nil)
	require.NoError(t, err)
	require.NotNil(t, inactiveKeyInfo)
	inactiveKeyInfo.IsActive = false

	tests := []struct {
		name    string
		apiKey  string
		wantErr bool
		wantKey *APIKeyInfo
	}{
		{
			name:    "valid API key",
			apiKey:  validKey,
			wantErr: false,
			wantKey: keyInfo,
		},
		{
			name:    "expired API key",
			apiKey:  expiredKey,
			wantErr: true,
			wantKey: nil,
		},
		{
			name:    "inactive API key",
			apiKey:  inactiveKey,
			wantErr: true,
			wantKey: nil,
		},
		{
			name:    "invalid API key format",
			apiKey:  "invalid-key-format",
			wantErr: true,
			wantKey: nil,
		},
		{
			name:    "non-existent API key",
			apiKey:  "usk_nonexistent1234567890abcdef",
			wantErr: true,
			wantKey: nil,
		},
		{
			name:    "empty API key",
			apiKey:  "",
			wantErr: true,
			wantKey: nil,
		},
		{
			name:    "API key without prefix",
			apiKey:  "abcd1234567890",
			wantErr: true,
			wantKey: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := manager.ValidateAPIKey(tt.apiKey)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.wantKey.ID, result.ID)
				assert.Equal(t, tt.wantKey.Name, result.Name)
				assert.Equal(t, tt.wantKey.UserID, result.UserID)
				assert.Equal(t, tt.wantKey.Permissions, result.Permissions)
				assert.NotNil(t, result.LastUsedAt) // Should be updated during validation
			}
		})
	}
}

func TestAPIKeyManager_RevokeAPIKey(t *testing.T) {
	manager := NewAPIKeyManager()
	
	// Generate API keys
	_, keyInfo1, err := manager.GenerateAPIKey("Key 1", "user-123", []string{APIKeyPermissions.ReadURLs}, nil)
	require.NoError(t, err)
	
	_, _, err = manager.GenerateAPIKey("Key 2", "user-456", []string{APIKeyPermissions.WriteURLs}, nil)
	require.NoError(t, err)

	tests := []struct {
		name    string
		keyID   string
		wantErr bool
	}{
		{
			name:    "revoke existing key",
			keyID:   keyInfo1.ID,
			wantErr: false,
		},
		{
			name:    "revoke non-existent key",
			keyID:   "non-existent-id",
			wantErr: true,
		},
		{
			name:    "revoke already revoked key",
			keyID:   keyInfo1.ID,
			wantErr: false, // Should not error, just mark as inactive
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.RevokeAPIKey(tt.keyID)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				
				// Verify key is inactive if it exists
				if tt.keyID == keyInfo1.ID {
					assert.False(t, keyInfo1.IsActive)
				}
			}
		})
	}
}

func TestAPIKeyManager_ListAPIKeys(t *testing.T) {
	manager := NewAPIKeyManager()
	
	// Generate API keys for different users
	_, _, err := manager.GenerateAPIKey("User1 Key1", "user-123", []string{APIKeyPermissions.ReadURLs}, nil)
	require.NoError(t, err)
	
	_, _, err = manager.GenerateAPIKey("User1 Key2", "user-123", []string{APIKeyPermissions.WriteURLs}, nil)
	require.NoError(t, err)
	
	_, _, err = manager.GenerateAPIKey("User2 Key1", "user-456", []string{APIKeyPermissions.AdminAccess}, nil)
	require.NoError(t, err)

	tests := []struct {
		name          string
		userID        string
		expectedCount int
	}{
		{
			name:          "list keys for user with 2 keys",
			userID:        "user-123",
			expectedCount: 2,
		},
		{
			name:          "list keys for user with 1 key",
			userID:        "user-456",
			expectedCount: 1,
		},
		{
			name:          "list keys for user with no keys",
			userID:        "user-789",
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keys := manager.ListAPIKeys(tt.userID)
			assert.Len(t, keys, tt.expectedCount)
			
			// Verify all keys belong to the correct user
			for _, key := range keys {
				assert.Equal(t, tt.userID, key.UserID)
			}
		})
	}
}

func TestAPIKeyInfo_HasPermission(t *testing.T) {
	tests := []struct {
		name        string
		permissions []string
		checkPerm   string
		want        bool
	}{
		{
			name:        "has specific permission",
			permissions: []string{APIKeyPermissions.ReadURLs, APIKeyPermissions.WriteURLs},
			checkPerm:   APIKeyPermissions.ReadURLs,
			want:        true,
		},
		{
			name:        "does not have permission",
			permissions: []string{APIKeyPermissions.ReadURLs},
			checkPerm:   APIKeyPermissions.WriteURLs,
			want:        false,
		},
		{
			name:        "has admin access (grants all permissions)",
			permissions: []string{APIKeyPermissions.AdminAccess},
			checkPerm:   APIKeyPermissions.DeleteURLs,
			want:        true,
		},
		{
			name:        "empty permissions",
			permissions: []string{},
			checkPerm:   APIKeyPermissions.ReadURLs,
			want:        false,
		},
		{
			name:        "nil permissions",
			permissions: nil,
			checkPerm:   APIKeyPermissions.ReadURLs,
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyInfo := &APIKeyInfo{
				Permissions: tt.permissions,
			}
			assert.Equal(t, tt.want, keyInfo.HasPermission(tt.checkPerm))
		})
	}
}

func TestAPIKeyInfo_IsExpired(t *testing.T) {
	now := time.Now()
	
	tests := []struct {
		name      string
		expiresAt *time.Time
		want      bool
	}{
		{
			name:      "not expired",
			expiresAt: timePtr(now.Add(time.Hour)),
			want:      false,
		},
		{
			name:      "expired",
			expiresAt: timePtr(now.Add(-time.Hour)),
			want:      true,
		},
		{
			name:      "no expiration",
			expiresAt: nil,
			want:      false,
		},
		{
			name:      "expires exactly now",
			expiresAt: timePtr(now),
			want:      true, // Should be considered expired if exactly at expiration time
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyInfo := &APIKeyInfo{
				ExpiresAt: tt.expiresAt,
			}
			assert.Equal(t, tt.want, keyInfo.IsExpired())
		})
	}
}

func TestAPIKeyManager_CreateAdminAPIKey(t *testing.T) {
	manager := NewAPIKeyManager()
	
	apiKey, err := manager.CreateAdminAPIKey()
	assert.NoError(t, err)
	assert.NotEmpty(t, apiKey)
	assert.Contains(t, apiKey, "usk_")
	
	// Validate the created key
	keyInfo, err := manager.ValidateAPIKey(apiKey)
	assert.NoError(t, err)
	assert.NotNil(t, keyInfo)
	assert.Equal(t, "Admin Key", keyInfo.Name)
	assert.Equal(t, "admin", keyInfo.UserID)
	assert.Contains(t, keyInfo.Permissions, APIKeyPermissions.AdminAccess)
	assert.True(t, keyInfo.HasPermission(APIKeyPermissions.AdminAccess))
	assert.True(t, keyInfo.HasPermission(APIKeyPermissions.ReadURLs)) // Admin should have all permissions
}

func TestAPIKeyManager_ConcurrentAccess(t *testing.T) {
	manager := NewAPIKeyManager()
	
	// Test concurrent key generation
	done := make(chan bool, 10)
	keys := make(chan string, 10)
	
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()

			apiKey, _, err := manager.GenerateAPIKey(
				fmt.Sprintf("Concurrent Key %d", id),
				fmt.Sprintf("user-%d", id),
				[]string{APIKeyPermissions.ReadURLs},
				nil,
			)
			
			if err == nil {
				keys <- apiKey
			}
		}(i)
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
	close(keys)
	
	// Verify all keys are unique
	keySet := make(map[string]bool)
	keyCount := 0
	for key := range keys {
		assert.False(t, keySet[key], "Duplicate key generated: %s", key)
		keySet[key] = true
		keyCount++
	}
	
	assert.Equal(t, 10, keyCount, "Expected 10 unique keys")
}

// Helper function to create time pointer
func timePtr(t time.Time) *time.Time {
	return &t
}
