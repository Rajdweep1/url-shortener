package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/rajweepmondal/url-shortener/internal/models"
	"github.com/rajweepmondal/url-shortener/internal/repository/interfaces"
)

// AnalyticsRepository implements the AnalyticsRepository interface for PostgreSQL
type AnalyticsRepository struct {
	db *sql.DB
}

// NewAnalyticsRepository creates a new PostgreSQL analytics repository
func NewAnalyticsRepository(db *sql.DB) interfaces.AnalyticsRepository {
	return &AnalyticsRepository{db: db}
}

// RecordAccess records an access event for analytics
func (r *AnalyticsRepository) RecordAccess(ctx context.Context, analytics *models.Analytics) error {
	query := `
		INSERT INTO analytics (id, short_code, accessed_at, ip_address, user_agent, 
							  referer, country_code, city, device_type)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	
	_, err := r.db.ExecContext(ctx, query,
		analytics.ID,
		analytics.ShortCode,
		analytics.AccessedAt,
		analytics.IPAddress,
		analytics.UserAgent,
		analytics.Referer,
		analytics.CountryCode,
		analytics.City,
		analytics.DeviceType,
	)
	
	if err != nil {
		return fmt.Errorf("failed to record access: %w", err)
	}
	
	return nil
}

// GetAnalytics retrieves analytics data for a URL
func (r *AnalyticsRepository) GetAnalytics(ctx context.Context, shortCode string, from, to time.Time) ([]*models.Analytics, error) {
	query := `
		SELECT id, short_code, accessed_at, ip_address, user_agent, 
			   referer, country_code, city, device_type
		FROM analytics 
		WHERE short_code = $1 AND accessed_at BETWEEN $2 AND $3
		ORDER BY accessed_at DESC
	`
	
	rows, err := r.db.QueryContext(ctx, query, shortCode, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to get analytics: %w", err)
	}
	defer rows.Close()
	
	var analytics []*models.Analytics
	for rows.Next() {
		a := &models.Analytics{}
		err := rows.Scan(
			&a.ID,
			&a.ShortCode,
			&a.AccessedAt,
			&a.IPAddress,
			&a.UserAgent,
			&a.Referer,
			&a.CountryCode,
			&a.City,
			&a.DeviceType,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan analytics: %w", err)
		}
		analytics = append(analytics, a)
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate analytics: %w", err)
	}
	
	return analytics, nil
}

// GetDailyStats retrieves daily statistics for a URL
func (r *AnalyticsRepository) GetDailyStats(ctx context.Context, shortCode string, days int) ([]*models.DailyStats, error) {
	query := `
		SELECT DATE(accessed_at) as date, COUNT(*) as clicks
		FROM analytics 
		WHERE short_code = $1 AND accessed_at >= NOW() - INTERVAL '%d days'
		GROUP BY DATE(accessed_at)
		ORDER BY date DESC
	`
	
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(query, days), shortCode)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily stats: %w", err)
	}
	defer rows.Close()
	
	var stats []*models.DailyStats
	for rows.Next() {
		s := &models.DailyStats{}
		err := rows.Scan(&s.Date, &s.Clicks)
		if err != nil {
			return nil, fmt.Errorf("failed to scan daily stats: %w", err)
		}
		stats = append(stats, s)
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate daily stats: %w", err)
	}
	
	return stats, nil
}

// GetURLStats retrieves comprehensive statistics for a URL
func (r *AnalyticsRepository) GetURLStats(ctx context.Context, shortCode string) (*models.URLStats, error) {
	// Get URL info
	urlQuery := `
		SELECT id, short_code, original_url, created_at, updated_at, 
			   click_count, last_accessed_at, custom_alias, user_id, expires_at, is_active
		FROM urls 
		WHERE short_code = $1
	`
	
	url := &models.URL{}
	err := r.db.QueryRowContext(ctx, urlQuery, shortCode).Scan(
		&url.ID,
		&url.ShortCode,
		&url.OriginalURL,
		&url.CreatedAt,
		&url.UpdatedAt,
		&url.ClickCount,
		&url.LastAccessedAt,
		&url.CustomAlias,
		&url.UserID,
		&url.ExpiresAt,
		&url.IsActive,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, models.ErrURLNotFound
		}
		return nil, fmt.Errorf("failed to get URL for stats: %w", err)
	}
	
	stats := &models.URLStats{
		URL:         url,
		TotalClicks: url.ClickCount,
	}
	
	// Get unique clicks (distinct IP addresses)
	uniqueQuery := `SELECT COUNT(DISTINCT ip_address) FROM analytics WHERE short_code = $1 AND ip_address IS NOT NULL`
	err = r.db.QueryRowContext(ctx, uniqueQuery, shortCode).Scan(&stats.UniqueClicks)
	if err != nil {
		return nil, fmt.Errorf("failed to get unique clicks: %w", err)
	}
	
	// Get clicks today
	todayQuery := `SELECT COUNT(*) FROM analytics WHERE short_code = $1 AND DATE(accessed_at) = CURRENT_DATE`
	err = r.db.QueryRowContext(ctx, todayQuery, shortCode).Scan(&stats.ClicksToday)
	if err != nil {
		return nil, fmt.Errorf("failed to get today's clicks: %w", err)
	}
	
	// Get clicks this week
	weekQuery := `SELECT COUNT(*) FROM analytics WHERE short_code = $1 AND accessed_at >= DATE_TRUNC('week', NOW())`
	err = r.db.QueryRowContext(ctx, weekQuery, shortCode).Scan(&stats.ClicksThisWeek)
	if err != nil {
		return nil, fmt.Errorf("failed to get this week's clicks: %w", err)
	}
	
	// Get top countries
	topCountries, err := r.GetTopCountries(ctx, shortCode, 5)
	if err != nil {
		return nil, fmt.Errorf("failed to get top countries: %w", err)
	}
	stats.TopCountries = topCountries
	
	// Get top referers
	topReferers, err := r.GetTopReferers(ctx, shortCode, 5)
	if err != nil {
		return nil, fmt.Errorf("failed to get top referers: %w", err)
	}
	stats.TopReferers = topReferers
	
	return stats, nil
}

// GetTopCountries retrieves top countries for a URL
func (r *AnalyticsRepository) GetTopCountries(ctx context.Context, shortCode string, limit int) ([]string, error) {
	query := `
		SELECT country_code, COUNT(*) as count
		FROM analytics 
		WHERE short_code = $1 AND country_code IS NOT NULL
		GROUP BY country_code
		ORDER BY count DESC
		LIMIT $2
	`
	
	rows, err := r.db.QueryContext(ctx, query, shortCode, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get top countries: %w", err)
	}
	defer rows.Close()
	
	var countries []string
	for rows.Next() {
		var country string
		var count int64
		err := rows.Scan(&country, &count)
		if err != nil {
			return nil, fmt.Errorf("failed to scan country: %w", err)
		}
		countries = append(countries, country)
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate countries: %w", err)
	}
	
	return countries, nil
}

// GetTopReferers retrieves top referers for a URL
func (r *AnalyticsRepository) GetTopReferers(ctx context.Context, shortCode string, limit int) ([]string, error) {
	query := `
		SELECT referer, COUNT(*) as count
		FROM analytics 
		WHERE short_code = $1 AND referer IS NOT NULL AND referer != ''
		GROUP BY referer
		ORDER BY count DESC
		LIMIT $2
	`
	
	rows, err := r.db.QueryContext(ctx, query, shortCode, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get top referers: %w", err)
	}
	defer rows.Close()
	
	var referers []string
	for rows.Next() {
		var referer string
		var count int64
		err := rows.Scan(&referer, &count)
		if err != nil {
			return nil, fmt.Errorf("failed to scan referer: %w", err)
		}
		referers = append(referers, referer)
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate referers: %w", err)
	}
	
	return referers, nil
}
