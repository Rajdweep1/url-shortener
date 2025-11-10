package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"

	"github.com/rajweepmondal/url-shortener/internal/models"
	"github.com/rajweepmondal/url-shortener/internal/service"
)

// URLHandler handles HTTP requests for URL operations
type URLHandler struct {
	urlService *service.URLService
	logger     *zap.Logger
}

// NewURLHandler creates a new HTTP URL handler
func NewURLHandler(urlService *service.URLService, logger *zap.Logger) *URLHandler {
	return &URLHandler{
		urlService: urlService,
		logger:     logger,
	}
}

// CreateShortURLRequest represents the request body for creating a short URL
type CreateShortURLRequest struct {
	OriginalURL   string  `json:"original_url" validate:"required,url"`
	CustomAlias   *string `json:"custom_alias,omitempty"`
	ExpiresInDays *int    `json:"expires_in_days,omitempty"`
	UserID        *string `json:"user_id,omitempty"`
}

// CreateShortURLResponse represents the response for creating a short URL
type CreateShortURLResponse struct {
	URL      *URLResponse `json:"url"`
	ShortURL string       `json:"short_url"`
}

// URLResponse represents a URL in HTTP responses
type URLResponse struct {
	ID             string     `json:"id"`
	ShortCode      string     `json:"short_code"`
	OriginalURL    string     `json:"original_url"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	ClickCount     int64      `json:"click_count"`
	LastAccessedAt *time.Time `json:"last_accessed_at,omitempty"`
	CustomAlias    *string    `json:"custom_alias,omitempty"`
	UserID         *string    `json:"user_id,omitempty"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	IsActive       bool       `json:"is_active"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// HealthResponse represents a health check response
type HealthResponse struct {
	Status       string            `json:"status"`
	Timestamp    time.Time         `json:"timestamp"`
	Version      string            `json:"version"`
	Dependencies map[string]string `json:"dependencies"`
}

// CreateShortURL handles POST /api/v1/urls
func (h *URLHandler) CreateShortURL(w http.ResponseWriter, r *http.Request) {
	// Limit request body size to 1MB
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB

	var req CreateShortURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Check for various body size limit errors
		errStr := err.Error()
		if errStr == "http: request body too large" ||
		   strings.Contains(errStr, "request body too large") ||
		   strings.Contains(errStr, "body too large") {
			h.writeErrorResponse(w, http.StatusBadRequest, "invalid_request", "Request body too large")
			return
		}
		h.writeErrorResponse(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	// Validate user_id length
	if req.UserID != nil && len(*req.UserID) > 255 {
		h.writeErrorResponse(w, http.StatusBadRequest, "invalid_request", "User ID cannot exceed 255 characters")
		return
	}

	// Convert to service request
	createReq := &models.CreateURLRequest{
		OriginalURL: req.OriginalURL,
		UserID:      req.UserID,
		CustomAlias: req.CustomAlias,
	}

	if req.ExpiresInDays != nil {
		if *req.ExpiresInDays < 0 {
			h.writeErrorResponse(w, http.StatusBadRequest, "invalid_request", "Expiration days cannot be negative")
			return
		}
		duration := time.Duration(*req.ExpiresInDays) * 24 * time.Hour
		createReq.ExpiresIn = &duration
	}

	// Call service
	url, shortURL, err := h.urlService.ShortenURL(r.Context(), createReq)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	// Return response
	response := &CreateShortURLResponse{
		URL:      h.urlToResponse(url),
		ShortURL: shortURL,
	}

	h.writeJSONResponse(w, http.StatusCreated, response)
}

// GetOriginalURL handles GET /{shortCode} - redirect to original URL
func (h *URLHandler) GetOriginalURL(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shortCode := vars["shortCode"]

	if shortCode == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "invalid_request", "Short code is required")
		return
	}

	// Extract client info
	clientInfo := h.extractClientInfo(r)

	// Call service
	originalURL, err := h.urlService.GetOriginalURL(r.Context(), shortCode, clientInfo)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	// Redirect to original URL
	http.Redirect(w, r, originalURL, http.StatusFound)
}

// GetURLInfo handles GET /api/v1/urls/{shortCode}
func (h *URLHandler) GetURLInfo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shortCode := vars["shortCode"]
	userID := r.URL.Query().Get("user_id")

	var userIDPtr *string
	if userID != "" {
		userIDPtr = &userID
	}

	url, err := h.urlService.GetURLInfo(r.Context(), shortCode, userIDPtr)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.writeJSONResponse(w, http.StatusOK, h.urlToResponse(url))
}

// ListURLs handles GET /api/v1/urls
func (h *URLHandler) ListURLs(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	pageSize := 10
	if ps := r.URL.Query().Get("page_size"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 && parsed <= 100 {
			pageSize = parsed
		}
	}

	pageToken := r.URL.Query().Get("page_token")
	userID := r.URL.Query().Get("user_id")

	var userIDPtr *string
	if userID != "" {
		userIDPtr = &userID
	}

	// Create request
	listReq := &models.ListURLsRequest{
		UserID:   userIDPtr,
		Page:     1,
		PageSize: pageSize,
		SortBy:   "created_at",
		SortDesc: true,
	}

	// Parse page token if provided
	if pageToken != "" {
		// Simple page token parsing - in production you'd use proper encoding
		// For now, assume page token is just the page number
		if page, err := strconv.Atoi(pageToken); err == nil && page > 0 {
			listReq.Page = page
		}
	}

	// Call service
	listResp, err := h.urlService.ListURLs(r.Context(), listReq)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	// Convert to response
	urlResponses := make([]*URLResponse, len(listResp.URLs))
	for i, url := range listResp.URLs {
		urlResponses[i] = h.urlToResponse(url)
	}

	// Generate next page token
	var nextPageToken string
	if listResp.Page < listResp.TotalPages {
		nextPageToken = strconv.Itoa(listResp.Page + 1)
	}

	response := map[string]interface{}{
		"urls":            urlResponses,
		"total_count":     listResp.TotalCount,
		"total_pages":     listResp.TotalPages,
		"current_page":    listResp.Page,
		"page_size":       listResp.PageSize,
		"next_page_token": nextPageToken,
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// UpdateURL handles PUT /api/v1/urls/{shortCode}
func (h *URLHandler) UpdateURL(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shortCode := vars["shortCode"]

	var req struct {
		OriginalURL *string    `json:"original_url,omitempty"`
		CustomAlias *string    `json:"custom_alias,omitempty"`
		ExpiresAt   *time.Time `json:"expires_at,omitempty"`
		IsActive    *bool      `json:"is_active,omitempty"`
		UserID      *string    `json:"user_id,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	// Convert to domain model
	updates := &models.URL{
		CustomAlias: req.CustomAlias,
		ExpiresAt:   req.ExpiresAt,
	}

	if req.OriginalURL != nil {
		updates.OriginalURL = *req.OriginalURL
	}

	// Call service
	url, err := h.urlService.UpdateURLWithActiveStatus(r.Context(), shortCode, updates, req.IsActive, req.UserID)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.writeJSONResponse(w, http.StatusOK, h.urlToResponse(url))
}

// DeleteURL handles DELETE /api/v1/urls/{shortCode}
func (h *URLHandler) DeleteURL(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shortCode := vars["shortCode"]
	userID := r.URL.Query().Get("user_id")

	var userIDPtr *string
	if userID != "" {
		userIDPtr = &userID
	}

	err := h.urlService.DeleteURL(r.Context(), shortCode, userIDPtr)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetAnalytics handles GET /api/v1/analytics/{shortCode}
func (h *URLHandler) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shortCode := vars["shortCode"]
	userID := r.URL.Query().Get("user_id")

	var userIDPtr *string
	if userID != "" {
		userIDPtr = &userID
	}

	// Default time range - last 30 days
	to := time.Now()
	from := to.AddDate(0, 0, -30)

	analytics, err := h.urlService.GetAnalytics(r.Context(), shortCode, from, to, userIDPtr)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.writeJSONResponse(w, http.StatusOK, analytics)
}

// RedirectURL handles URL redirection
func (h *URLHandler) RedirectURL(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shortCode := vars["shortCode"]

	if shortCode == "" {
		http.Error(w, "Short code is required", http.StatusBadRequest)
		return
	}

	// Get client information for analytics
	clientInfo := h.getClientInfo(r)

	// Get original URL and track click
	originalURL, err := h.urlService.GetOriginalURL(r.Context(), shortCode, clientInfo)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	// Redirect to original URL
	http.Redirect(w, r, originalURL, http.StatusFound)
}

// getClientInfo extracts client information from the HTTP request
func (h *URLHandler) getClientInfo(r *http.Request) *service.ClientInfo {
	clientInfo := &service.ClientInfo{}

	// Extract IP address
	ip := h.getClientIP(r)
	if ip != "" {
		clientInfo.IPAddress = &ip
	}

	// Extract User-Agent
	if userAgent := r.Header.Get("User-Agent"); userAgent != "" {
		clientInfo.UserAgent = &userAgent
	}

	// Extract Referer
	if referer := r.Header.Get("Referer"); referer != "" {
		clientInfo.Referer = &referer
	}

	return clientInfo
}

// getClientIP extracts the client IP address from the request
func (h *URLHandler) getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fall back to RemoteAddr
	if idx := strings.LastIndex(r.RemoteAddr, ":"); idx != -1 {
		return r.RemoteAddr[:idx]
	}
	return r.RemoteAddr
}

// GetHealth handles GET /api/v1/health
func (h *URLHandler) GetHealth(w http.ResponseWriter, r *http.Request) {
	response := &HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   "1.0.0",
		Dependencies: map[string]string{
			"database": "healthy",
			"cache":    "healthy",
		},
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// Helper methods

func (h *URLHandler) urlToResponse(url *models.URL) *URLResponse {
	return &URLResponse{
		ID:             url.ID.String(),
		ShortCode:      url.ShortCode,
		OriginalURL:    url.OriginalURL,
		CreatedAt:      url.CreatedAt,
		UpdatedAt:      url.UpdatedAt,
		ClickCount:     url.ClickCount,
		LastAccessedAt: url.LastAccessedAt,
		CustomAlias:    url.CustomAlias,
		UserID:         url.UserID,
		ExpiresAt:      url.ExpiresAt,
		IsActive:       url.IsActive,
	}
}

func (h *URLHandler) extractClientInfo(r *http.Request) *service.ClientInfo {
	clientInfo := &service.ClientInfo{}

	// Extract IP address
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.Header.Get("X-Real-IP")
	}
	if ip == "" {
		ip = strings.Split(r.RemoteAddr, ":")[0]
	}
	if ip != "" {
		clientInfo.IPAddress = &ip
	}

	// Extract user agent
	userAgent := r.Header.Get("User-Agent")
	if userAgent != "" {
		clientInfo.UserAgent = &userAgent
	}

	// Extract referer
	referer := r.Header.Get("Referer")
	if referer != "" {
		clientInfo.Referer = &referer
	}

	return clientInfo
}

func (h *URLHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", zap.Error(err))
	}
}

func (h *URLHandler) writeErrorResponse(w http.ResponseWriter, statusCode int, errorCode, message string) {
	response := &ErrorResponse{
		Error:   errorCode,
		Message: message,
		Code:    statusCode,
	}
	h.writeJSONResponse(w, statusCode, response)
}

func (h *URLHandler) handleServiceError(w http.ResponseWriter, err error) {
	// Check if it's an AppError first
	if appErr, ok := err.(*models.AppError); ok {
		// Use the HTTP status from the AppError
		h.writeErrorResponse(w, appErr.HTTPStatus, string(appErr.Code), appErr.Message)
		return
	}

	// Fallback to string matching for other errors
	switch {
	case strings.Contains(err.Error(), "not found"):
		h.writeErrorResponse(w, http.StatusNotFound, "not_found", err.Error())
	case strings.Contains(err.Error(), "already exists"):
		h.writeErrorResponse(w, http.StatusConflict, "conflict", err.Error())
	case strings.Contains(err.Error(), "invalid"):
		h.writeErrorResponse(w, http.StatusBadRequest, "invalid_request", err.Error())
	case strings.Contains(err.Error(), "expired"):
		h.writeErrorResponse(w, http.StatusGone, "expired", err.Error())
	default:
		h.logger.Error("Service error", zap.Error(err))
		h.writeErrorResponse(w, http.StatusInternalServerError, "internal_error", "Internal server error")
	}
}
