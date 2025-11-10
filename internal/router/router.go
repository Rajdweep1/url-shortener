package router

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"

	httpHandler "github.com/rajweepmondal/url-shortener/internal/handler/http"
	"github.com/rajweepmondal/url-shortener/internal/middleware"
	"github.com/rajweepmondal/url-shortener/internal/service"
	"github.com/rajweepmondal/url-shortener/pkg/ratelimiter"
)

// Router holds the HTTP router and dependencies
type Router struct {
	router      *mux.Router
	urlHandler  *httpHandler.URLHandler
	authHandler *httpHandler.AuthHandler
	logger      *zap.Logger
}

// New creates a new HTTP router with all routes and middleware
func New(urlService *service.URLService, logger *zap.Logger, rateLimitMiddleware *ratelimiter.Middleware, authMiddleware *middleware.AuthMiddleware) *Router {
	r := &Router{
		router:      mux.NewRouter(),
		urlHandler:  httpHandler.NewURLHandler(urlService, logger),
		authHandler: httpHandler.NewAuthHandler(authMiddleware.GetAuthManager(), logger),
		logger:      logger,
	}

	r.setupMiddleware(rateLimitMiddleware, authMiddleware)
	r.setupRoutes()

	return r
}

// Handler returns the HTTP handler
func (r *Router) Handler() http.Handler {
	return r.router
}

// setupMiddleware configures global middleware
func (r *Router) setupMiddleware(rateLimitMiddleware *ratelimiter.Middleware, authMiddleware *middleware.AuthMiddleware) {
	// Apply middleware in order (first applied = outermost)
	r.router.Use(middleware.HTTPRecoveryMiddleware(r.logger))
	r.router.Use(middleware.HTTPLoggingMiddleware(r.logger))
	r.router.Use(middleware.HTTPCORSMiddleware())
	r.router.Use(middleware.HTTPSecurityMiddleware())
	r.router.Use(middleware.HTTPContentTypeMiddleware())
	r.router.Use(middleware.HTTPTimeoutMiddleware(30 * time.Second))
	r.router.Use(middleware.HTTPValidationMiddleware())
	r.router.Use(middleware.HTTPRateLimitMiddleware(rateLimitMiddleware))
	r.router.Use(authMiddleware.HTTPAuthMiddleware())
}

// setupRoutes configures all HTTP routes
func (r *Router) setupRoutes() {
	// Health check endpoints
	r.router.HandleFunc("/health", r.urlHandler.GetHealth).Methods("GET")
	r.router.HandleFunc("/api/v1/health", r.urlHandler.GetHealth).Methods("GET")

	// API v1 routes
	api := r.router.PathPrefix("/api/v1").Subrouter()

	// Admin-only authentication endpoints (for API key management)
	api.HandleFunc("/auth/api-keys", r.authHandler.CreateAPIKey).Methods("POST")

	// URL management endpoints
	api.HandleFunc("/urls", r.urlHandler.CreateShortURL).Methods("POST")
	api.HandleFunc("/urls", r.urlHandler.ListURLs).Methods("GET")
	api.HandleFunc("/urls/{shortCode}", r.urlHandler.GetURLInfo).Methods("GET")
	api.HandleFunc("/urls/{shortCode}", r.urlHandler.UpdateURL).Methods("PUT")
	api.HandleFunc("/urls/{shortCode}", r.urlHandler.DeleteURL).Methods("DELETE")

	// Analytics endpoint
	api.HandleFunc("/analytics/{shortCode}", r.urlHandler.GetAnalytics).Methods("GET")

	// Redirect endpoint (must be last to avoid conflicts)
	r.router.HandleFunc("/{shortCode}", r.urlHandler.RedirectURL).Methods("GET")

	// Add OPTIONS handler for CORS preflight
	r.router.Methods("OPTIONS").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Add a root handler for documentation
	r.router.HandleFunc("/", r.handleRoot).Methods("GET")
}

// handleRoot provides API documentation at the root path
func (r *Router) handleRoot(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	response := map[string]interface{}{
		"service": "URL Shortener API",
		"version": "1.0.0",
		"endpoints": map[string]interface{}{
			"health": map[string]string{
				"GET /health":           "Service health check",
				"GET /api/v1/health":    "Detailed health check",
			},
			"urls": map[string]string{
				"POST /api/v1/urls":              "Create a short URL (public)",
				"GET /api/v1/urls":               "List URLs (public)",
				"GET /api/v1/urls/{shortCode}":   "Get URL information (public)",
				"PUT /api/v1/urls/{shortCode}":   "Update URL (public)",
				"DELETE /api/v1/urls/{shortCode}": "Delete URL (public)",
			},
			"analytics": map[string]string{
				"GET /api/v1/analytics/{shortCode}": "Get URL analytics (admin only)",
			},
			"admin": map[string]string{
				"POST /api/v1/auth/api-keys": "Create API key (admin only)",
			},
			"redirect": map[string]string{
				"GET /{shortCode}": "Redirect to original URL (public)",
			},
		},
		"documentation": map[string]string{
			"postman_collection": "/api/v1/docs/postman",
			"openapi_spec":       "/api/v1/docs/openapi",
		},
		"protocols": []string{"HTTP/REST", "gRPC"},
		"grpc": map[string]interface{}{
			"port": 8080,
			"reflection": true,
			"services": []string{
				"url_shortener.v1.URLShortenerService",
			},
		},
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		r.logger.Error("Failed to encode root response", zap.Error(err))
	}
}

// GetRoutes returns a list of all registered routes for debugging
func (r *Router) GetRoutes() []string {
	var routes []string
	
	err := r.router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		template, err := route.GetPathTemplate()
		if err != nil {
			return nil
		}
		
		methods, err := route.GetMethods()
		if err != nil {
			methods = []string{"*"}
		}
		
		for _, method := range methods {
			routes = append(routes, method+" "+template)
		}
		
		return nil
	})
	
	if err != nil {
		r.logger.Error("Failed to walk routes", zap.Error(err))
	}
	
	return routes
}
