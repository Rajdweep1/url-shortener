package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rajweepmondal/url-shortener/internal/models"
	"github.com/rajweepmondal/url-shortener/internal/repository/interfaces"
	"github.com/rajweepmondal/url-shortener/pkg/shortener"
	"github.com/rajweepmondal/url-shortener/pkg/validator"
)

// URLService handles business logic for URL operations
type URLService struct {
	urlRepo       interfaces.URLRepository
	analyticsRepo interfaces.AnalyticsRepository
	cacheRepo     interfaces.CacheRepository
	shortener     *shortener.Shortener
	validator     *validator.URLValidator
	baseURL       string
	cacheTTL      time.Duration
}

// NewURLService creates a new URL service
func NewURLService(
	urlRepo interfaces.URLRepository,
	analyticsRepo interfaces.AnalyticsRepository,
	cacheRepo interfaces.CacheRepository,
	shortCodeLength int,
	baseURL string,
	cacheTTL time.Duration,
) *URLService {
	return &URLService{
		urlRepo:       urlRepo,
		analyticsRepo: analyticsRepo,
		cacheRepo:     cacheRepo,
		shortener:     shortener.New(shortCodeLength),
		validator:     validator.NewURLValidator(),
		baseURL:       baseURL,
		cacheTTL:      cacheTTL,
	}
}

// ShortenURL creates a new shortened URL
func (s *URLService) ShortenURL(ctx context.Context, req *models.CreateURLRequest) (*models.URL, string, error) {
	// Validate the original URL
	if err := s.validator.ValidateURL(req.OriginalURL); err != nil {
		return nil, "", models.ErrValidation(err.Error())
	}

	// Validate custom alias if provided
	if req.CustomAlias != nil {
		if err := validator.ValidateCustomAlias(*req.CustomAlias); err != nil {
			return nil, "", models.ErrValidation(err.Error())
		}
	}

	// Check if URL already exists (idempotency)
	existingURL, err := s.urlRepo.GetByOriginalURL(ctx, req.OriginalURL, req.UserID)
	if err == nil && existingURL != nil && !existingURL.IsExpired() {
		// Return existing URL
		shortURL := fmt.Sprintf("%s/%s", s.baseURL, existingURL.ShortCode)
		return existingURL, shortURL, nil
	}

	// Create new URL
	url := &models.URL{
		ID:          uuid.New(),
		OriginalURL: req.OriginalURL,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		ClickCount:  0,
		UserID:      req.UserID,
		IsActive:    true,
	}

	// Set expiration if provided
	if req.ExpiresIn != nil {
		expiresAt := time.Now().Add(*req.ExpiresIn)
		url.ExpiresAt = &expiresAt
	}

	// Generate short code
	var shortCode string
	if req.CustomAlias != nil {
		// Use custom alias
		customCode, err := s.shortener.GenerateCustomCode(*req.CustomAlias)
		if err != nil {
			return nil, "", models.ErrValidation(err.Error())
		}
		shortCode = customCode
		url.CustomAlias = req.CustomAlias
	} else {
		// Generate short code with collision handling
		for attempt := 0; attempt < 10; attempt++ {
			code, err := s.shortener.GenerateWithCollisionHandling(req.OriginalURL, attempt)
			if err != nil {
				return nil, "", models.ErrInternal("failed to generate short code")
			}

			// Check if code already exists
			_, err = s.urlRepo.GetByShortCode(ctx, code)
			if err != nil {
				if err == models.ErrURLNotFound {
					shortCode = code
					break
				}
				return nil, "", models.ErrInternal("failed to check short code uniqueness")
			}
		}

		if shortCode == "" {
			return nil, "", models.ErrInternal("failed to generate unique short code")
		}
	}

	url.ShortCode = shortCode

	// Save to database
	if err := s.urlRepo.Create(ctx, url); err != nil {
		return nil, "", err
	}

	// Cache the URL
	if err := s.cacheURL(ctx, url); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Failed to cache URL: %v\n", err)
	}

	shortURL := fmt.Sprintf("%s/%s", s.baseURL, url.ShortCode)
	return url, shortURL, nil
}

// GetOriginalURL retrieves the original URL for redirection
func (s *URLService) GetOriginalURL(ctx context.Context, shortCode string, clientInfo *ClientInfo) (string, error) {
	// Validate short code format (allow both generated codes and custom aliases)
	if !s.isValidShortCodeOrAlias(shortCode) {
		return "", models.ErrInvalidShortCode
	}

	// Try to get from cache first, but we still need to check expiration from database
	cachedURL, cacheErr := s.getCachedURL(ctx, shortCode)
	if cacheErr == nil && cachedURL != "" {
		// We have a cached URL, but we still need to check if it's expired
		// Get the URL from database to check expiration and active status
		url, err := s.urlRepo.GetByShortCode(ctx, shortCode)
		if err != nil {
			// If URL not found in DB, invalidate cache and return error
			go s.invalidateCache(context.Background(), shortCode)
			return "", err
		}

		// Check if URL is expired
		if url.IsExpired() {
			// Invalidate cache for expired URL
			go s.invalidateCache(context.Background(), shortCode)
			return "", models.ErrURLExpired
		}

		// Check if URL is active
		if !url.IsActive {
			// Invalidate cache for inactive URL
			go s.invalidateCache(context.Background(), shortCode)
			return "", models.ErrURLInactive
		}

		// URL is valid, use cached version and record analytics
		go s.recordAccess(context.Background(), shortCode, clientInfo)
		go func() {
			bgCtx := context.Background()
			s.urlRepo.IncrementClickCount(bgCtx, shortCode)
			s.urlRepo.UpdateLastAccessed(bgCtx, shortCode)
		}()
		return cachedURL, nil
	}

	// Get from database
	url, err := s.urlRepo.GetByShortCode(ctx, shortCode)
	if err != nil {
		return "", err
	}

	// Check if URL is expired
	if url.IsExpired() {
		return "", models.ErrURLExpired
	}

	// Check if URL is active
	if !url.IsActive {
		return "", models.ErrURLInactive
	}

	// Update click count and last accessed time
	go func() {
		bgCtx := context.Background()
		s.urlRepo.IncrementClickCount(bgCtx, shortCode)
		s.urlRepo.UpdateLastAccessed(bgCtx, shortCode)
	}()

	// Record analytics
	go s.recordAccess(context.Background(), shortCode, clientInfo)

	// Cache the URL
	go s.cacheURL(context.Background(), url)

	return url.OriginalURL, nil
}

// GetURLInfo retrieves detailed information about a URL
func (s *URLService) GetURLInfo(ctx context.Context, shortCode string, userID *string) (*models.URL, error) {
	url, err := s.urlRepo.GetByShortCode(ctx, shortCode)
	if err != nil {
		return nil, err
	}

	// Check user authorization if userID is provided
	if userID != nil && url.UserID != nil && *url.UserID != *userID {
		return nil, models.ErrForbidden("access denied")
	}

	return url, nil
}

// ListURLs retrieves a paginated list of URLs
func (s *URLService) ListURLs(ctx context.Context, req *models.ListURLsRequest) (*models.ListURLsResponse, error) {
	// Validate pagination parameters
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}

	// Validate sort parameters
	validSortFields := map[string]bool{
		"created_at":       true,
		"click_count":      true,
		"last_accessed_at": true,
	}
	if !validSortFields[req.SortBy] {
		req.SortBy = "created_at"
	}

	urls, totalCount, err := s.urlRepo.List(ctx, req)
	if err != nil {
		return nil, err
	}

	totalPages := int((totalCount + int64(req.PageSize) - 1) / int64(req.PageSize))

	return &models.ListURLsResponse{
		URLs:       urls,
		TotalCount: totalCount,
		TotalPages: totalPages,
		Page:       req.Page,
		PageSize:   req.PageSize,
	}, nil
}

// UpdateURL updates an existing URL
func (s *URLService) UpdateURL(ctx context.Context, shortCode string, updates *models.URL, userID *string) (*models.URL, error) {
	// Get existing URL
	existingURL, err := s.urlRepo.GetByShortCode(ctx, shortCode)
	if err != nil {
		return nil, err
	}

	// Check user authorization
	if userID != nil && existingURL.UserID != nil && *existingURL.UserID != *userID {
		return nil, models.ErrForbidden("access denied")
	}

	// Validate new original URL if provided
	if updates.OriginalURL != "" && updates.OriginalURL != existingURL.OriginalURL {
		if err := s.validator.ValidateURL(updates.OriginalURL); err != nil {
			return nil, models.ErrValidation(err.Error())
		}
		existingURL.OriginalURL = updates.OriginalURL
	}

	// Update custom alias if provided
	if updates.CustomAlias != nil {
		if err := validator.ValidateCustomAlias(*updates.CustomAlias); err != nil {
			return nil, models.ErrValidation(err.Error())
		}
		existingURL.CustomAlias = updates.CustomAlias
	}

	// Update expiration if provided
	if updates.ExpiresAt != nil {
		existingURL.ExpiresAt = updates.ExpiresAt
	}

	// Update active status if provided
	existingURL.IsActive = updates.IsActive

	// Save changes
	if err := s.urlRepo.Update(ctx, existingURL); err != nil {
		return nil, err
	}

	// Invalidate cache synchronously to ensure consistency
	s.invalidateCache(ctx, shortCode)

	return existingURL, nil
}

// DeleteURL soft deletes a URL
func (s *URLService) DeleteURL(ctx context.Context, shortCode string, userID *string) error {
	// Check if URL exists and user has permission
	url, err := s.urlRepo.GetByShortCode(ctx, shortCode)
	if err != nil {
		return err
	}

	// Check user authorization
	if userID != nil && url.UserID != nil && *url.UserID != *userID {
		return models.ErrForbidden("access denied")
	}

	// Soft delete
	if err := s.urlRepo.Delete(ctx, shortCode, userID); err != nil {
		return err
	}

	// Invalidate cache synchronously to ensure consistency
	s.invalidateCache(ctx, shortCode)

	return nil
}

// GetAnalytics retrieves analytics for a URL
func (s *URLService) GetAnalytics(ctx context.Context, shortCode string, from, to time.Time, userID *string) (*models.URLStats, error) {
	// Check if URL exists and user has permission
	url, err := s.urlRepo.GetByShortCode(ctx, shortCode)
	if err != nil {
		return nil, err
	}

	// Check user authorization
	if userID != nil && url.UserID != nil && *url.UserID != *userID {
		return nil, models.ErrForbidden("access denied")
	}

	// Get comprehensive statistics
	stats, err := s.analyticsRepo.GetURLStats(ctx, shortCode)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

// Helper methods

// isValidShortCodeOrAlias validates both generated short codes and custom aliases
func (s *URLService) isValidShortCodeOrAlias(code string) bool {
	// Check if it's a valid generated short code (base62)
	if shortener.IsValidShortCode(code) {
		return true
	}

	// Check if it's a valid custom alias (alphanumeric + dashes + underscores)
	if len(code) < 3 || len(code) > 50 {
		return false
	}

	for _, char := range code {
		if !((char >= '0' && char <= '9') ||
			(char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			char == '-' || char == '_') {
			return false
		}
	}

	return true
}

// cacheURL caches a URL in Redis
func (s *URLService) cacheURL(ctx context.Context, url *models.URL) error {
	return s.cacheRepo.Set(ctx, 
		fmt.Sprintf("url:%s", url.ShortCode), 
		url.OriginalURL, 
		s.cacheTTL,
	)
}

// getCachedURL retrieves a cached URL
func (s *URLService) getCachedURL(ctx context.Context, shortCode string) (string, error) {
	return s.cacheRepo.Get(ctx, fmt.Sprintf("url:%s", shortCode))
}

// invalidateCache removes a URL from cache
func (s *URLService) invalidateCache(ctx context.Context, shortCode string) {
	s.cacheRepo.Delete(ctx, fmt.Sprintf("url:%s", shortCode))
}

// recordAccess records an access event for analytics
func (s *URLService) recordAccess(ctx context.Context, shortCode string, clientInfo *ClientInfo) {
	if clientInfo == nil {
		return
	}

	analytics := &models.Analytics{
		ID:          uuid.New(),
		ShortCode:   shortCode,
		AccessedAt:  time.Now(),
		IPAddress:   clientInfo.IPAddress,
		UserAgent:   clientInfo.UserAgent,
		Referer:     clientInfo.Referer,
		CountryCode: clientInfo.CountryCode,
		City:        clientInfo.City,
		DeviceType:  clientInfo.DeviceType,
	}

	if err := s.analyticsRepo.RecordAccess(ctx, analytics); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Failed to record analytics: %v\n", err)
	}
}

// ClientInfo contains client information for analytics
type ClientInfo struct {
	IPAddress   *string
	UserAgent   *string
	Referer     *string
	CountryCode *string
	City        *string
	DeviceType  *string
}
