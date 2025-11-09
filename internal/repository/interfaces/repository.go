package interfaces

import (
	"context"
	"time"

	"github.com/rajweepmondal/url-shortener/internal/models"
)

// URLRepository defines the interface for URL data operations
type URLRepository interface {
	// Create creates a new shortened URL
	Create(ctx context.Context, url *models.URL) error
	
	// GetByShortCode retrieves a URL by its short code
	GetByShortCode(ctx context.Context, shortCode string) (*models.URL, error)
	
	// GetByOriginalURL retrieves a URL by its original URL (for idempotency)
	GetByOriginalURL(ctx context.Context, originalURL string, userID *string) (*models.URL, error)
	
	// GetByID retrieves a URL by its ID
	GetByID(ctx context.Context, id string) (*models.URL, error)
	
	// Update updates an existing URL
	Update(ctx context.Context, url *models.URL) error
	
	// Delete soft deletes a URL (sets is_active to false)
	Delete(ctx context.Context, shortCode string, userID *string) error
	
	// IncrementClickCount increments the click count for a URL
	IncrementClickCount(ctx context.Context, shortCode string) error
	
	// UpdateLastAccessed updates the last accessed timestamp
	UpdateLastAccessed(ctx context.Context, shortCode string) error
	
	// List retrieves URLs with pagination and filtering
	List(ctx context.Context, req *models.ListURLsRequest) ([]*models.URL, int64, error)
	
	// GetExpiredURLs retrieves URLs that have expired
	GetExpiredURLs(ctx context.Context, limit int) ([]*models.URL, error)
	
	// CleanupExpiredURLs removes expired URLs
	CleanupExpiredURLs(ctx context.Context) (int64, error)
}

// AnalyticsRepository defines the interface for analytics data operations
type AnalyticsRepository interface {
	// RecordAccess records an access event for analytics
	RecordAccess(ctx context.Context, analytics *models.Analytics) error
	
	// GetAnalytics retrieves analytics data for a URL
	GetAnalytics(ctx context.Context, shortCode string, from, to time.Time) ([]*models.Analytics, error)
	
	// GetDailyStats retrieves daily statistics for a URL
	GetDailyStats(ctx context.Context, shortCode string, days int) ([]*models.DailyStats, error)
	
	// GetURLStats retrieves comprehensive statistics for a URL
	GetURLStats(ctx context.Context, shortCode string) (*models.URLStats, error)
	
	// GetTopCountries retrieves top countries for a URL
	GetTopCountries(ctx context.Context, shortCode string, limit int) ([]string, error)
	
	// GetTopReferers retrieves top referers for a URL
	GetTopReferers(ctx context.Context, shortCode string, limit int) ([]string, error)
}

// CacheRepository defines the interface for caching operations
type CacheRepository interface {
	// Get retrieves a value from cache
	Get(ctx context.Context, key string) (string, error)
	
	// Set stores a value in cache with expiration
	Set(ctx context.Context, key string, value string, expiration time.Duration) error
	
	// Delete removes a value from cache
	Delete(ctx context.Context, key string) error
	
	// Exists checks if a key exists in cache
	Exists(ctx context.Context, key string) (bool, error)
	
	// Increment increments a counter in cache
	Increment(ctx context.Context, key string) (int64, error)
	
	// IncrementWithExpiry increments a counter with expiration
	IncrementWithExpiry(ctx context.Context, key string, expiration time.Duration) (int64, error)
	
	// GetMultiple retrieves multiple values from cache
	GetMultiple(ctx context.Context, keys []string) (map[string]string, error)
	
	// SetMultiple stores multiple values in cache
	SetMultiple(ctx context.Context, values map[string]string, expiration time.Duration) error
	
	// FlushAll clears all cache entries
	FlushAll(ctx context.Context) error
}

// RateLimitRepository defines the interface for rate limiting operations
type RateLimitRepository interface {
	// CheckRateLimit checks if a request is within rate limits
	CheckRateLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, int, error)
	
	// IncrementRateLimit increments the rate limit counter
	IncrementRateLimit(ctx context.Context, key string, window time.Duration) (int, error)
	
	// GetRateLimitInfo gets current rate limit information
	GetRateLimitInfo(ctx context.Context, key string) (int, time.Duration, error)
	
	// ResetRateLimit resets the rate limit for a key
	ResetRateLimit(ctx context.Context, key string) error
}
