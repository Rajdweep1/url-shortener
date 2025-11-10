package auth

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// AuthManager combines JWT and API key authentication
type AuthManager struct {
	jwtManager    *JWTManager
	apiKeyManager *APIKeyManager
}

// AuthContext contains authentication information
type AuthContext struct {
	UserID      string   `json:"user_id"`
	Username    string   `json:"username"`
	Email       string   `json:"email"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	AuthType    string   `json:"auth_type"` // "jwt" or "api_key"
	IsAdmin     bool     `json:"is_admin"`
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	JWTSecret       string        `envconfig:"JWT_SECRET" required:"true"`
	JWTDuration     time.Duration `envconfig:"JWT_DURATION" default:"24h"`
	JWTIssuer       string        `envconfig:"JWT_ISSUER" default:"url-shortener"`
	APIKeyRequired  bool          `envconfig:"API_KEY_REQUIRED" default:"true"`
	AdminAPIKey     string        `envconfig:"ADMIN_API_KEY"`
	EnableJWT       bool          `envconfig:"ENABLE_JWT" default:"true"`
	EnableAPIKey    bool          `envconfig:"ENABLE_API_KEY" default:"true"`
}

// NewAuthManager creates a new authentication manager
func NewAuthManager(config AuthConfig) (*AuthManager, error) {
	var jwtManager *JWTManager
	var apiKeyManager *APIKeyManager

	// Initialize JWT manager if enabled
	if config.EnableJWT {
		if config.JWTSecret == "" {
			return nil, fmt.Errorf("JWT secret is required when JWT is enabled")
		}
		jwtManager = NewJWTManager(config.JWTSecret, config.JWTDuration, config.JWTIssuer)
	}

	// Initialize API key manager if enabled
	if config.EnableAPIKey {
		apiKeyManager = NewAPIKeyManager()
		
		// Create admin API key if provided
		if config.AdminAPIKey != "" {
			// For simplicity, we'll store the admin key directly
			// In production, this should be properly hashed and stored
			adminKeyInfo := &APIKeyInfo{
				ID:          "admin",
				Name:        "Admin API Key",
				HashedKey:   hashString(config.AdminAPIKey),
				Permissions: []string{APIKeyPermissions.AdminAccess},
				CreatedAt:   time.Now(),
				IsActive:    true,
				UserID:      "admin",
			}
			apiKeyManager.keys[adminKeyInfo.HashedKey] = adminKeyInfo
		}
	}

	return &AuthManager{
		jwtManager:    jwtManager,
		apiKeyManager: apiKeyManager,
	}, nil
}

// AuthenticateToken authenticates a token (JWT or API key) and returns auth context
func (am *AuthManager) AuthenticateToken(token string) (*AuthContext, error) {
	// Try API key authentication first (for both usk_ prefixed keys and admin keys)
	if am.apiKeyManager != nil {
		if strings.HasPrefix(token, "usk_") {
			return am.authenticateAPIKey(token)
		}

		// Try admin API key (simple string match for demo)
		if hashedToken := hashString(token); am.apiKeyManager.keys[hashedToken] != nil {
			return am.authenticateAPIKey(token)
		}
	}

	// Try JWT authentication
	if am.jwtManager != nil {
		return am.authenticateJWT(token)
	}

	return nil, fmt.Errorf("no authentication methods enabled")
}

// authenticateJWT authenticates a JWT token
func (am *AuthManager) authenticateJWT(tokenString string) (*AuthContext, error) {
	claims, err := am.jwtManager.ValidateToken(tokenString)
	if err != nil {
		return nil, fmt.Errorf("JWT validation failed: %w", err)
	}

	return &AuthContext{
		UserID:   claims.UserID,
		Username: claims.Username,
		Email:    claims.Email,
		Roles:    claims.Roles,
		AuthType: "jwt",
		IsAdmin:  claims.IsAdmin(),
	}, nil
}

// authenticateAPIKey authenticates an API key
func (am *AuthManager) authenticateAPIKey(apiKey string) (*AuthContext, error) {
	keyInfo, err := am.apiKeyManager.ValidateAPIKey(apiKey)
	if err != nil {
		return nil, fmt.Errorf("API key validation failed: %w", err)
	}

	return &AuthContext{
		UserID:      keyInfo.UserID,
		Username:    keyInfo.Name,
		Permissions: keyInfo.Permissions,
		AuthType:    "api_key",
		IsAdmin:     keyInfo.HasPermission(APIKeyPermissions.AdminAccess),
	}, nil
}

// GenerateJWT generates a new JWT token
func (am *AuthManager) GenerateJWT(userID, username, email string, roles []string) (string, error) {
	if am.jwtManager == nil {
		return "", fmt.Errorf("JWT authentication is not enabled")
	}
	return am.jwtManager.GenerateToken(userID, username, email, roles)
}

// GenerateAPIKey generates a new API key
func (am *AuthManager) GenerateAPIKey(name, userID string, permissions []string, expiresAt *time.Time) (string, *APIKeyInfo, error) {
	if am.apiKeyManager == nil {
		return "", nil, fmt.Errorf("API key authentication is not enabled")
	}
	return am.apiKeyManager.GenerateAPIKey(name, userID, permissions, expiresAt)
}

// RequirePermission checks if the auth context has a specific permission
func (ac *AuthContext) RequirePermission(permission string) error {
	if ac.IsAdmin {
		return nil // Admins have all permissions
	}

	for _, p := range ac.Permissions {
		if p == permission {
			return nil
		}
	}

	return fmt.Errorf("insufficient permissions: required %s", permission)
}

// RequireRole checks if the auth context has a specific role
func (ac *AuthContext) RequireRole(role string) error {
	if ac.IsAdmin && role != "super_admin" {
		return nil // Admins have most roles except super_admin
	}

	for _, r := range ac.Roles {
		if r == role {
			return nil
		}
	}

	return fmt.Errorf("insufficient role: required %s", role)
}

// Context keys for storing auth information
type contextKey string

const (
	AuthContextKey contextKey = "auth_context"
)

// WithAuthContext adds auth context to the context
func WithAuthContext(ctx context.Context, authCtx *AuthContext) context.Context {
	return context.WithValue(ctx, AuthContextKey, authCtx)
}

// FromContext extracts auth context from the context
func FromContext(ctx context.Context) (*AuthContext, bool) {
	authCtx, ok := ctx.Value(AuthContextKey).(*AuthContext)
	return authCtx, ok
}

// hashString creates a simple hash of a string (for demo purposes)
func hashString(s string) string {
	// In production, use proper cryptographic hashing
	return fmt.Sprintf("hash_%s", s)
}
