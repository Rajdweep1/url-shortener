package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"
)

// APIKeyManager handles API key operations
type APIKeyManager struct {
	keys map[string]*APIKeyInfo // In production, this would be stored in database
	mu   sync.RWMutex
}

// APIKeyInfo contains information about an API key
type APIKeyInfo struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	HashedKey   string    `json:"hashed_key"`
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	IsActive    bool      `json:"is_active"`
	UserID      string    `json:"user_id"`
}

// APIKeyPermissions defines available permissions
var APIKeyPermissions = struct {
	ReadURLs      string
	WriteURLs     string
	DeleteURLs    string
	ReadAnalytics string
	AdminAccess   string
}{
	ReadURLs:      "urls:read",
	WriteURLs:     "urls:write", 
	DeleteURLs:    "urls:delete",
	ReadAnalytics: "analytics:read",
	AdminAccess:   "admin:access",
}

// NewAPIKeyManager creates a new API key manager
func NewAPIKeyManager() *APIKeyManager {
	return &APIKeyManager{
		keys: make(map[string]*APIKeyInfo),
	}
}

// GenerateAPIKey generates a new API key
func (m *APIKeyManager) GenerateAPIKey(name, userID string, permissions []string, expiresAt *time.Time) (string, *APIKeyInfo, error) {
	// Generate random key
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return "", nil, fmt.Errorf("failed to generate random key: %w", err)
	}

	// Create readable key with prefix
	rawKey := fmt.Sprintf("usk_%s", base64.URLEncoding.EncodeToString(keyBytes))
	
	// Hash the key for storage
	hasher := sha256.New()
	hasher.Write([]byte(rawKey))
	hashedKey := hex.EncodeToString(hasher.Sum(nil))

	// Generate unique ID
	idBytes := make([]byte, 8)
	if _, err := rand.Read(idBytes); err != nil {
		return "", nil, fmt.Errorf("failed to generate key ID: %w", err)
	}
	keyID := hex.EncodeToString(idBytes)

	keyInfo := &APIKeyInfo{
		ID:          keyID,
		Name:        name,
		HashedKey:   hashedKey,
		Permissions: permissions,
		CreatedAt:   time.Now(),
		ExpiresAt:   expiresAt,
		IsActive:    true,
		UserID:      userID,
	}

	// Store the key info (in production, this would be in database)
	m.mu.Lock()
	m.keys[hashedKey] = keyInfo
	m.mu.Unlock()

	return rawKey, keyInfo, nil
}

// ValidateAPIKey validates an API key and returns the key info
func (m *APIKeyManager) ValidateAPIKey(apiKey string) (*APIKeyInfo, error) {
	// Check if key has correct prefix (allow admin keys without prefix)
	if !strings.HasPrefix(apiKey, "usk_") {
		// Check if it's a direct admin key by hashing and looking up
		hashedKey := hashString(apiKey)
		m.mu.RLock()
		keyInfo, exists := m.keys[hashedKey]
		m.mu.RUnlock()
		if !exists {
			return nil, fmt.Errorf("invalid API key format")
		}

		// Validate admin key
		if keyInfo.IsExpired() {
			return nil, fmt.Errorf("API key has expired")
		}

		if !keyInfo.IsActive {
			return nil, fmt.Errorf("API key is inactive")
		}

		// Update last used time
		now := time.Now()
		keyInfo.LastUsedAt = &now

		return keyInfo, nil
	}

	// Hash the provided key
	hasher := sha256.New()
	hasher.Write([]byte(apiKey))
	hashedKey := hex.EncodeToString(hasher.Sum(nil))

	// Look up the key
	m.mu.RLock()
	keyInfo, exists := m.keys[hashedKey]
	m.mu.RUnlock()
	if !exists {
		return nil, fmt.Errorf("API key not found")
	}

	// Check if key is active
	if !keyInfo.IsActive {
		return nil, fmt.Errorf("API key is inactive")
	}

	// Check if key is expired
	if keyInfo.ExpiresAt != nil && time.Now().After(*keyInfo.ExpiresAt) {
		return nil, fmt.Errorf("API key has expired")
	}

	// Update last used time
	now := time.Now()
	keyInfo.LastUsedAt = &now

	return keyInfo, nil
}

// RevokeAPIKey revokes an API key
func (m *APIKeyManager) RevokeAPIKey(keyID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, keyInfo := range m.keys {
		if keyInfo.ID == keyID {
			keyInfo.IsActive = false
			return nil
		}
	}
	return fmt.Errorf("API key not found")
}

// ListAPIKeys returns all API keys for a user
func (m *APIKeyManager) ListAPIKeys(userID string) []*APIKeyInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var keys []*APIKeyInfo
	for _, keyInfo := range m.keys {
		if keyInfo.UserID == userID {
			keys = append(keys, keyInfo)
		}
	}
	return keys
}

// HasPermission checks if an API key has a specific permission
func (k *APIKeyInfo) HasPermission(permission string) bool {
	for _, p := range k.Permissions {
		if p == permission || p == APIKeyPermissions.AdminAccess {
			return true
		}
	}
	return false
}

// IsExpired checks if the API key is expired
func (k *APIKeyInfo) IsExpired() bool {
	return k.ExpiresAt != nil && time.Now().After(*k.ExpiresAt)
}

// CreateAdminAPIKey creates a default admin API key for development/testing
func (m *APIKeyManager) CreateAdminAPIKey() (string, error) {
	adminPermissions := []string{
		APIKeyPermissions.ReadURLs,
		APIKeyPermissions.WriteURLs,
		APIKeyPermissions.DeleteURLs,
		APIKeyPermissions.ReadAnalytics,
		APIKeyPermissions.AdminAccess,
	}

	// Create a key that expires in 1 year
	expiresAt := time.Now().AddDate(1, 0, 0)
	
	apiKey, _, err := m.GenerateAPIKey("Admin Key", "admin", adminPermissions, &expiresAt)
	if err != nil {
		return "", err
	}

	return apiKey, nil
}
