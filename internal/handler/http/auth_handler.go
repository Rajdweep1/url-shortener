package http

import (
	"encoding/json"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/rajweepmondal/url-shortener/pkg/auth"
)

// AuthHandler handles authentication-related HTTP requests
type AuthHandler struct {
	authManager *auth.AuthManager
	logger      *zap.Logger
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(authManager *auth.AuthManager, logger *zap.Logger) *AuthHandler {
	return &AuthHandler{
		authManager: authManager,
		logger:      logger,
	}
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      UserInfo  `json:"user"`
}

// UserInfo represents user information
type UserInfo struct {
	ID       string   `json:"id"`
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Roles    []string `json:"roles"`
}

// CreateAPIKeyRequest represents an API key creation request
type CreateAPIKeyRequest struct {
	Name        string    `json:"name" validate:"required"`
	Permissions []string  `json:"permissions"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// CreateAPIKeyResponse represents an API key creation response
type CreateAPIKeyResponse struct {
	APIKey  string           `json:"api_key"`
	KeyInfo *auth.APIKeyInfo `json:"key_info"`
}

// Login handles user login and JWT token generation
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// In a real implementation, you would:
	// 1. Validate credentials against database
	// 2. Check if user is active
	// 3. Handle password hashing/verification
	// 4. Implement rate limiting for login attempts
	
	// For demo purposes, we'll accept admin/admin credentials
	if req.Username != "admin" || req.Password != "admin" {
		h.writeErrorResponse(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Generate JWT token
	roles := []string{"admin"}
	token, err := h.authManager.GenerateJWT("admin", req.Username, "admin@example.com", roles)
	if err != nil {
		h.logger.Error("Failed to generate JWT token", zap.Error(err))
		h.writeErrorResponse(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	response := LoginResponse{
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour), // Should match JWT duration
		User: UserInfo{
			ID:       "admin",
			Username: req.Username,
			Email:    "admin@example.com",
			Roles:    roles,
		},
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// CreateAPIKey handles API key creation
func (h *AuthHandler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	// Get auth context from middleware
	authCtx, ok := auth.FromContext(r.Context())
	if !ok {
		h.writeErrorResponse(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Only admins can create API keys
	if !authCtx.IsAdmin {
		h.writeErrorResponse(w, "Admin access required", http.StatusForbidden)
		return
	}

	var req CreateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Set default permissions if none provided
	if len(req.Permissions) == 0 {
		req.Permissions = []string{
			auth.APIKeyPermissions.ReadURLs,
			auth.APIKeyPermissions.WriteURLs,
		}
	}

	// Generate API key
	apiKey, keyInfo, err := h.authManager.GenerateAPIKey(req.Name, authCtx.UserID, req.Permissions, req.ExpiresAt)
	if err != nil {
		h.logger.Error("Failed to generate API key", zap.Error(err))
		h.writeErrorResponse(w, "Failed to generate API key", http.StatusInternalServerError)
		return
	}

	response := CreateAPIKeyResponse{
		APIKey:  apiKey,
		KeyInfo: keyInfo,
	}

	h.writeJSONResponse(w, http.StatusCreated, response)
}

// ValidateToken handles token validation
func (h *AuthHandler) ValidateToken(w http.ResponseWriter, r *http.Request) {
	// Extract token from request
	token := h.extractToken(r)
	if token == "" {
		h.writeErrorResponse(w, "Token required", http.StatusBadRequest)
		return
	}

	// Validate token
	authCtx, err := h.authManager.AuthenticateToken(token)
	if err != nil {
		h.writeErrorResponse(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// Return token information
	response := map[string]interface{}{
		"valid":        true,
		"user_id":      authCtx.UserID,
		"username":     authCtx.Username,
		"email":        authCtx.Email,
		"roles":        authCtx.Roles,
		"permissions":  authCtx.Permissions,
		"auth_type":    authCtx.AuthType,
		"is_admin":     authCtx.IsAdmin,
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// GetProfile returns the current user's profile
func (h *AuthHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	// Get auth context from middleware
	authCtx, ok := auth.FromContext(r.Context())
	if !ok {
		h.writeErrorResponse(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	profile := UserInfo{
		ID:       authCtx.UserID,
		Username: authCtx.Username,
		Email:    authCtx.Email,
		Roles:    authCtx.Roles,
	}

	h.writeJSONResponse(w, http.StatusOK, profile)
}

// extractToken extracts the authentication token from HTTP request
func (h *AuthHandler) extractToken(r *http.Request) string {
	// Try Authorization header first
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			return authHeader[7:]
		}
		if len(authHeader) > 7 && authHeader[:7] == "ApiKey " {
			return authHeader[7:]
		}
		return authHeader
	}

	// Try X-API-Key header
	apiKey := r.Header.Get("X-API-Key")
	if apiKey != "" {
		return apiKey
	}

	// Try query parameter (less secure)
	return r.URL.Query().Get("token")
}

// writeJSONResponse writes a JSON response
func (h *AuthHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", zap.Error(err))
	}
}

// writeErrorResponse writes a JSON error response
func (h *AuthHandler) writeErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	response := map[string]interface{}{
		"error": map[string]interface{}{
			"message": message,
			"code":    statusCode,
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode error response", zap.Error(err))
	}
}
