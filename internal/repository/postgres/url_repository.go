package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/rajweepmondal/url-shortener/internal/models"
	"github.com/rajweepmondal/url-shortener/internal/repository/interfaces"
)

// URLRepository implements the URLRepository interface for PostgreSQL
type URLRepository struct {
	db *sql.DB
}

// NewURLRepository creates a new PostgreSQL URL repository
func NewURLRepository(db *sql.DB) interfaces.URLRepository {
	return &URLRepository{db: db}
}

// Create creates a new shortened URL
func (r *URLRepository) Create(ctx context.Context, url *models.URL) error {
	query := `
		INSERT INTO urls (id, short_code, original_url, created_at, updated_at, 
						 click_count, custom_alias, user_id, expires_at, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	
	_, err := r.db.ExecContext(ctx, query,
		url.ID,
		url.ShortCode,
		url.OriginalURL,
		url.CreatedAt,
		url.UpdatedAt,
		url.ClickCount,
		url.CustomAlias,
		url.UserID,
		url.ExpiresAt,
		url.IsActive,
	)
	
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code {
			case "23505": // unique_violation
				if pqErr.Constraint == "urls_short_code_key" {
					return models.ErrShortCodeExists
				}
				if pqErr.Constraint == "idx_urls_custom_alias" {
					return models.ErrCustomAliasExists
				}
			}
		}
		return fmt.Errorf("failed to create URL: %w", err)
	}
	
	return nil
}

// GetByShortCode retrieves a URL by its short code
func (r *URLRepository) GetByShortCode(ctx context.Context, shortCode string) (*models.URL, error) {
	query := `
		SELECT id, short_code, original_url, created_at, updated_at, 
			   click_count, last_accessed_at, custom_alias, user_id, expires_at, is_active
		FROM urls 
		WHERE short_code = $1 AND is_active = true
	`
	
	url := &models.URL{}
	err := r.db.QueryRowContext(ctx, query, shortCode).Scan(
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
		return nil, fmt.Errorf("failed to get URL by short code: %w", err)
	}
	
	// Check if URL has expired
	if url.IsExpired() {
		return nil, models.ErrURLExpired
	}
	
	return url, nil
}

// GetByOriginalURL retrieves a URL by its original URL (for idempotency)
func (r *URLRepository) GetByOriginalURL(ctx context.Context, originalURL string, userID *string) (*models.URL, error) {
	query := `
		SELECT id, short_code, original_url, created_at, updated_at, 
			   click_count, last_accessed_at, custom_alias, user_id, expires_at, is_active
		FROM urls 
		WHERE original_url = $1 AND is_active = true
	`
	args := []interface{}{originalURL}
	
	// Add user_id condition if provided
	if userID != nil {
		query += " AND user_id = $2"
		args = append(args, *userID)
	} else {
		query += " AND user_id IS NULL"
	}
	
	query += " ORDER BY created_at DESC LIMIT 1"
	
	url := &models.URL{}
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
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
		return nil, fmt.Errorf("failed to get URL by original URL: %w", err)
	}
	
	return url, nil
}

// GetByID retrieves a URL by its ID
func (r *URLRepository) GetByID(ctx context.Context, id string) (*models.URL, error) {
	urlID, err := uuid.Parse(id)
	if err != nil {
		return nil, models.ErrBadRequest("invalid URL ID format")
	}
	
	query := `
		SELECT id, short_code, original_url, created_at, updated_at, 
			   click_count, last_accessed_at, custom_alias, user_id, expires_at, is_active
		FROM urls 
		WHERE id = $1
	`
	
	url := &models.URL{}
	err = r.db.QueryRowContext(ctx, query, urlID).Scan(
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
		return nil, fmt.Errorf("failed to get URL by ID: %w", err)
	}
	
	return url, nil
}

// Update updates an existing URL
func (r *URLRepository) Update(ctx context.Context, url *models.URL) error {
	query := `
		UPDATE urls 
		SET original_url = $2, custom_alias = $3, expires_at = $4, is_active = $5, updated_at = NOW()
		WHERE short_code = $1
	`
	
	result, err := r.db.ExecContext(ctx, query,
		url.ShortCode,
		url.OriginalURL,
		url.CustomAlias,
		url.ExpiresAt,
		url.IsActive,
	)
	
	if err != nil {
		return fmt.Errorf("failed to update URL: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return models.ErrURLNotFound
	}
	
	return nil
}

// Delete soft deletes a URL (sets is_active to false)
func (r *URLRepository) Delete(ctx context.Context, shortCode string, userID *string) error {
	query := `UPDATE urls SET is_active = false, updated_at = NOW() WHERE short_code = $1`
	args := []interface{}{shortCode}
	
	// Add user_id condition if provided
	if userID != nil {
		query += " AND user_id = $2"
		args = append(args, *userID)
	}
	
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete URL: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return models.ErrURLNotFound
	}
	
	return nil
}

// IncrementClickCount increments the click count for a URL
func (r *URLRepository) IncrementClickCount(ctx context.Context, shortCode string) error {
	query := `UPDATE urls SET click_count = click_count + 1 WHERE short_code = $1 AND is_active = true`
	
	result, err := r.db.ExecContext(ctx, query, shortCode)
	if err != nil {
		return fmt.Errorf("failed to increment click count: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return models.ErrURLNotFound
	}
	
	return nil
}

// UpdateLastAccessed updates the last accessed timestamp
func (r *URLRepository) UpdateLastAccessed(ctx context.Context, shortCode string) error {
	query := `UPDATE urls SET last_accessed_at = NOW() WHERE short_code = $1 AND is_active = true`

	_, err := r.db.ExecContext(ctx, query, shortCode)
	if err != nil {
		return fmt.Errorf("failed to update last accessed: %w", err)
	}

	return nil
}

// List retrieves URLs with pagination and filtering
func (r *URLRepository) List(ctx context.Context, req *models.ListURLsRequest) ([]*models.URL, int64, error) {
	// Build the base query
	baseQuery := `FROM urls WHERE is_active = true`
	args := []interface{}{}
	argIndex := 1

	// Add user filter if provided
	if req.UserID != nil {
		baseQuery += fmt.Sprintf(" AND user_id = $%d", argIndex)
		args = append(args, *req.UserID)
		argIndex++
	}

	// Get total count
	countQuery := "SELECT COUNT(*) " + baseQuery
	var totalCount int64
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get total count: %w", err)
	}

	// Build the main query with sorting and pagination
	sortColumn := "created_at"
	switch req.SortBy {
	case "click_count":
		sortColumn = "click_count"
	case "last_accessed_at":
		sortColumn = "last_accessed_at"
	}

	sortOrder := "ASC"
	if req.SortDesc {
		sortOrder = "DESC"
	}

	query := fmt.Sprintf(`
		SELECT id, short_code, original_url, created_at, updated_at,
			   click_count, last_accessed_at, custom_alias, user_id, expires_at, is_active
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, baseQuery, sortColumn, sortOrder, argIndex, argIndex+1)

	offset := (req.Page - 1) * req.PageSize
	args = append(args, req.PageSize, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list URLs: %w", err)
	}
	defer rows.Close()

	var urls []*models.URL
	for rows.Next() {
		url := &models.URL{}
		err := rows.Scan(
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
			return nil, 0, fmt.Errorf("failed to scan URL: %w", err)
		}
		urls = append(urls, url)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to iterate URLs: %w", err)
	}

	return urls, totalCount, nil
}

// GetExpiredURLs retrieves URLs that have expired
func (r *URLRepository) GetExpiredURLs(ctx context.Context, limit int) ([]*models.URL, error) {
	query := `
		SELECT id, short_code, original_url, created_at, updated_at,
			   click_count, last_accessed_at, custom_alias, user_id, expires_at, is_active
		FROM urls
		WHERE expires_at IS NOT NULL AND expires_at < NOW() AND is_active = true
		ORDER BY expires_at ASC
		LIMIT $1
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get expired URLs: %w", err)
	}
	defer rows.Close()

	var urls []*models.URL
	for rows.Next() {
		url := &models.URL{}
		err := rows.Scan(
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
			return nil, fmt.Errorf("failed to scan expired URL: %w", err)
		}
		urls = append(urls, url)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate expired URLs: %w", err)
	}

	return urls, nil
}

// CleanupExpiredURLs removes expired URLs
func (r *URLRepository) CleanupExpiredURLs(ctx context.Context) (int64, error) {
	query := `UPDATE urls SET is_active = false WHERE expires_at IS NOT NULL AND expires_at < NOW() AND is_active = true`

	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired URLs: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}
