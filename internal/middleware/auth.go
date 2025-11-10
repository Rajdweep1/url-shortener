package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/rajweepmondal/url-shortener/pkg/auth"
)

// AuthMiddleware handles authentication for HTTP requests
type AuthMiddleware struct {
	authManager *auth.AuthManager
	logger      *zap.Logger
	requireAuth bool
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(authManager *auth.AuthManager, logger *zap.Logger, requireAuth bool) *AuthMiddleware {
	return &AuthMiddleware{
		authManager: authManager,
		logger:      logger,
		requireAuth: requireAuth,
	}
}

// GetAuthManager returns the auth manager instance
func (am *AuthMiddleware) GetAuthManager() *auth.AuthManager {
	return am.authManager
}

// HTTPAuthMiddleware provides authentication for HTTP endpoints
func (am *AuthMiddleware) HTTPAuthMiddleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			am.logger.Info("Auth middleware called", zap.String("path", r.URL.Path), zap.String("method", r.Method), zap.Bool("requireAuth", am.requireAuth))

			// Skip auth for public endpoints
			if am.isPublicEndpoint(r) {
				am.logger.Info("Skipping auth for public endpoint", zap.String("path", r.URL.Path), zap.String("method", r.Method))
				next.ServeHTTP(w, r)
				return
			}

			am.logger.Info("Endpoint requires authentication", zap.String("path", r.URL.Path), zap.String("method", r.Method))

			// For admin endpoints, always require authentication regardless of requireAuth flag
			// For other protected endpoints, respect the requireAuth flag
			if !am.requireAuth && !am.isAdminEndpoint(r) {
				next.ServeHTTP(w, r)
				return
			}

			// Extract token from request
			token := am.extractToken(r)
			if token == "" {
				am.writeErrorResponse(w, "Authorization token required", http.StatusUnauthorized)
				return
			}

			// Authenticate token
			authCtx, err := am.authManager.AuthenticateToken(token)
			if err != nil {
				am.logger.Warn("Authentication failed", zap.Error(err), zap.String("path", r.URL.Path))
				am.writeErrorResponse(w, "Invalid or expired token", http.StatusUnauthorized)
				return
			}

			// Check permissions for admin endpoints
			if am.isAdminEndpoint(r) && !authCtx.IsAdmin {
				am.writeErrorResponse(w, "Admin access required", http.StatusForbidden)
				return
			}

			// Add auth context to request context
			ctx := auth.WithAuthContext(r.Context(), authCtx)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GRPCAuthInterceptor provides authentication for gRPC requests
func (am *AuthMiddleware) GRPCAuthInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Skip auth for public endpoints
		if am.isPublicGRPCEndpoint(info.FullMethod) {
			return handler(ctx, req)
		}

		// Skip auth if not required (for development)
		if !am.requireAuth {
			return handler(ctx, req)
		}

		// Extract token from metadata
		token, err := am.extractGRPCToken(ctx)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "Authorization token required")
		}

		// Authenticate token
		authCtx, err := am.authManager.AuthenticateToken(token)
		if err != nil {
			am.logger.Warn("gRPC Authentication failed", zap.Error(err), zap.String("method", info.FullMethod))
			return nil, status.Error(codes.Unauthenticated, "Invalid or expired token")
		}

		// Check permissions for admin endpoints
		if am.isAdminGRPCEndpoint(info.FullMethod) && !authCtx.IsAdmin {
			return nil, status.Error(codes.PermissionDenied, "Admin access required")
		}

		// Add auth context to request context
		ctx = auth.WithAuthContext(ctx, authCtx)
		return handler(ctx, req)
	}
}

// extractToken extracts the authentication token from HTTP request
func (am *AuthMiddleware) extractToken(r *http.Request) string {
	// Try Authorization header first (Bearer token or API key)
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		// Handle Bearer token
		if strings.HasPrefix(authHeader, "Bearer ") {
			return strings.TrimPrefix(authHeader, "Bearer ")
		}
		// Handle API key
		if strings.HasPrefix(authHeader, "ApiKey ") {
			return strings.TrimPrefix(authHeader, "ApiKey ")
		}
		// Handle direct token
		return authHeader
	}

	// Try X-API-Key header
	apiKey := r.Header.Get("X-API-Key")
	if apiKey != "" {
		return apiKey
	}

	// Try query parameter (less secure, for development only)
	return r.URL.Query().Get("api_key")
}

// extractGRPCToken extracts the authentication token from gRPC metadata
func (am *AuthMiddleware) extractGRPCToken(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "metadata not found")
	}

	// Try authorization header
	if values := md.Get("authorization"); len(values) > 0 {
		authHeader := values[0]
		if strings.HasPrefix(authHeader, "Bearer ") {
			return strings.TrimPrefix(authHeader, "Bearer "), nil
		}
		if strings.HasPrefix(authHeader, "ApiKey ") {
			return strings.TrimPrefix(authHeader, "ApiKey "), nil
		}
		return authHeader, nil
	}

	// Try x-api-key header
	if values := md.Get("x-api-key"); len(values) > 0 {
		return values[0], nil
	}

	return "", status.Error(codes.Unauthenticated, "authorization token not found")
}

// isPublicEndpoint checks if an HTTP endpoint is public
func (am *AuthMiddleware) isPublicEndpoint(r *http.Request) bool {
	publicPaths := []string{
		"/health",
		"/api/v1/health",
		"/",
	}

	// Check if this is a redirect endpoint (GET /{shortCode})
	if r.Method == "GET" && !strings.HasPrefix(r.URL.Path, "/api/") && r.URL.Path != "/" {
		am.logger.Debug("Public endpoint: redirect", zap.String("path", r.URL.Path))
		return true
	}

	// Core URL operations are now public (matches requirements)
	if strings.HasPrefix(r.URL.Path, "/api/v1/urls") {
		// All URL CRUD operations are public
		am.logger.Info("Public endpoint: URL operation", zap.String("path", r.URL.Path))
		return true
	}

	for _, path := range publicPaths {
		if r.URL.Path == path {
			am.logger.Debug("Public endpoint: static path", zap.String("path", r.URL.Path))
			return true
		}
	}

	am.logger.Debug("Protected endpoint", zap.String("path", r.URL.Path))
	return false
}

// isAdminEndpoint checks if an HTTP endpoint requires admin access
func (am *AuthMiddleware) isAdminEndpoint(r *http.Request) bool {
	// Analytics endpoints always require admin access
	if strings.HasPrefix(r.URL.Path, "/api/v1/analytics") {
		return true
	}

	// Admin authentication endpoints
	if strings.HasPrefix(r.URL.Path, "/api/v1/auth/") {
		// Only api-keys endpoint requires admin access
		return strings.HasSuffix(r.URL.Path, "/api-keys")
	}

	return false
}

// isPublicGRPCEndpoint checks if a gRPC endpoint is public
func (am *AuthMiddleware) isPublicGRPCEndpoint(method string) bool {
	publicMethods := []string{
		"/url_shortener.v1.URLShortenerService/GetHealthCheck",
		"/url_shortener.v1.URLShortenerService/GetOriginalURL",
		"/url_shortener.v1.URLShortenerService/CreateShortURL",
		"/url_shortener.v1.URLShortenerService/GetURLInfo",
		"/url_shortener.v1.URLShortenerService/ListURLs",
		"/url_shortener.v1.URLShortenerService/UpdateURL",
		"/url_shortener.v1.URLShortenerService/DeleteURL",
	}

	for _, publicMethod := range publicMethods {
		if method == publicMethod {
			return true
		}
	}

	return false
}

// isAdminGRPCEndpoint checks if a gRPC endpoint requires admin access
func (am *AuthMiddleware) isAdminGRPCEndpoint(method string) bool {
	adminMethods := []string{
		"/url_shortener.v1.URLShortenerService/GetAnalytics",
	}

	for _, adminMethod := range adminMethods {
		if method == adminMethod {
			return true
		}
	}

	return false
}

// writeErrorResponse writes a JSON error response
func (am *AuthMiddleware) writeErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	response := map[string]interface{}{
		"error": map[string]interface{}{
			"message": message,
			"code":    statusCode,
		},
		"timestamp": "2024-01-01T00:00:00Z", // In production, use actual timestamp
	}
	
	json.NewEncoder(w).Encode(response)
}
