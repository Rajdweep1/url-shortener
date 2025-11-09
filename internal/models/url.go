package models

import (
	"time"

	"github.com/google/uuid"
)

// URL represents a shortened URL in the system
type URL struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	ShortCode      string     `json:"short_code" db:"short_code"`
	OriginalURL    string     `json:"original_url" db:"original_url"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
	ClickCount     int64      `json:"click_count" db:"click_count"`
	LastAccessedAt *time.Time `json:"last_accessed_at,omitempty" db:"last_accessed_at"`
	CustomAlias    *string    `json:"custom_alias,omitempty" db:"custom_alias"`
	UserID         *string    `json:"user_id,omitempty" db:"user_id"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	IsActive       bool       `json:"is_active" db:"is_active"`
}

// IsExpired checks if the URL has expired
func (u *URL) IsExpired() bool {
	if u.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*u.ExpiresAt)
}

// Analytics represents analytics data for a URL
type Analytics struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	ShortCode    string     `json:"short_code" db:"short_code"`
	AccessedAt   time.Time  `json:"accessed_at" db:"accessed_at"`
	IPAddress    *string    `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent    *string    `json:"user_agent,omitempty" db:"user_agent"`
	Referer      *string    `json:"referer,omitempty" db:"referer"`
	CountryCode  *string    `json:"country_code,omitempty" db:"country_code"`
	City         *string    `json:"city,omitempty" db:"city"`
	DeviceType   *string    `json:"device_type,omitempty" db:"device_type"`
}

// CreateURLRequest represents a request to create a shortened URL
type CreateURLRequest struct {
	OriginalURL   string        `json:"original_url" validate:"required,url"`
	CustomAlias   *string       `json:"custom_alias,omitempty" validate:"omitempty,alphanum,min=3,max=50"`
	ExpiresIn     *time.Duration `json:"expires_in,omitempty"`
	UserID        *string       `json:"user_id,omitempty"`
}

// URLStats represents statistics for a URL
type URLStats struct {
	URL            *URL      `json:"url"`
	TotalClicks    int64     `json:"total_clicks"`
	UniqueClicks   int64     `json:"unique_clicks"`
	ClicksToday    int64     `json:"clicks_today"`
	ClicksThisWeek int64     `json:"clicks_this_week"`
	TopCountries   []string  `json:"top_countries"`
	TopReferers    []string  `json:"top_referers"`
}

// DailyStats represents daily statistics
type DailyStats struct {
	Date   time.Time `json:"date"`
	Clicks int64     `json:"clicks"`
}

// ListURLsRequest represents a request to list URLs
type ListURLsRequest struct {
	UserID   *string `json:"user_id,omitempty"`
	Page     int     `json:"page" validate:"min=1"`
	PageSize int     `json:"page_size" validate:"min=1,max=100"`
	SortBy   string  `json:"sort_by" validate:"oneof=created_at click_count last_accessed_at"`
	SortDesc bool    `json:"sort_desc"`
}

// ListURLsResponse represents a response for listing URLs
type ListURLsResponse struct {
	URLs       []*URL `json:"urls"`
	TotalCount int64  `json:"total_count"`
	TotalPages int    `json:"total_pages"`
	Page       int    `json:"page"`
	PageSize   int    `json:"page_size"`
}
