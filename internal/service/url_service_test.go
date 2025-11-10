package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/rajweepmondal/url-shortener/internal/models"
)

// Mock repositories for testing
type MockURLRepository struct {
	mock.Mock
}

func (m *MockURLRepository) Create(ctx context.Context, url *models.URL) error {
	args := m.Called(ctx, url)
	return args.Error(0)
}

func (m *MockURLRepository) GetByShortCode(ctx context.Context, shortCode string) (*models.URL, error) {
	args := m.Called(ctx, shortCode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.URL), args.Error(1)
}

func (m *MockURLRepository) GetByOriginalURL(ctx context.Context, originalURL string, userID *string) (*models.URL, error) {
	args := m.Called(ctx, originalURL, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.URL), args.Error(1)
}

func (m *MockURLRepository) Update(ctx context.Context, url *models.URL) error {
	args := m.Called(ctx, url)
	return args.Error(0)
}

func (m *MockURLRepository) Delete(ctx context.Context, shortCode string, userID *string) error {
	args := m.Called(ctx, shortCode, userID)
	return args.Error(0)
}

func (m *MockURLRepository) List(ctx context.Context, req *models.ListURLsRequest) ([]*models.URL, int64, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*models.URL), args.Get(1).(int64), args.Error(2)
}

func (m *MockURLRepository) IncrementClickCount(ctx context.Context, shortCode string) error {
	args := m.Called(ctx, shortCode)
	return args.Error(0)
}

func (m *MockURLRepository) UpdateLastAccessed(ctx context.Context, shortCode string) error {
	args := m.Called(ctx, shortCode)
	return args.Error(0)
}

func (m *MockURLRepository) GetByID(ctx context.Context, id string) (*models.URL, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.URL), args.Error(1)
}

func (m *MockURLRepository) GetExpiredURLs(ctx context.Context, limit int) ([]*models.URL, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.URL), args.Error(1)
}

func (m *MockURLRepository) CleanupExpiredURLs(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

type MockAnalyticsRepository struct {
	mock.Mock
}

func (m *MockAnalyticsRepository) RecordAccess(ctx context.Context, analytics *models.Analytics) error {
	args := m.Called(ctx, analytics)
	return args.Error(0)
}

func (m *MockAnalyticsRepository) GetURLStats(ctx context.Context, shortCode string) (*models.URLStats, error) {
	args := m.Called(ctx, shortCode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.URLStats), args.Error(1)
}

func (m *MockAnalyticsRepository) GetAnalytics(ctx context.Context, shortCode string, from, to time.Time) ([]*models.Analytics, error) {
	args := m.Called(ctx, shortCode, from, to)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Analytics), args.Error(1)
}

func (m *MockAnalyticsRepository) GetDailyStats(ctx context.Context, shortCode string, days int) ([]*models.DailyStats, error) {
	args := m.Called(ctx, shortCode, days)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.DailyStats), args.Error(1)
}

func (m *MockAnalyticsRepository) GetTopCountries(ctx context.Context, shortCode string, limit int) ([]string, error) {
	args := m.Called(ctx, shortCode, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockAnalyticsRepository) GetTopReferers(ctx context.Context, shortCode string, limit int) ([]string, error) {
	args := m.Called(ctx, shortCode, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

type MockCacheRepository struct {
	mock.Mock
}

func (m *MockCacheRepository) Get(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *MockCacheRepository) Set(ctx context.Context, key string, value string, expiration time.Duration) error {
	args := m.Called(ctx, key, value, expiration)
	return args.Error(0)
}

func (m *MockCacheRepository) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockCacheRepository) Exists(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

func (m *MockCacheRepository) Increment(ctx context.Context, key string) (int64, error) {
	args := m.Called(ctx, key)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockCacheRepository) IncrementWithExpiry(ctx context.Context, key string, expiration time.Duration) (int64, error) {
	args := m.Called(ctx, key, expiration)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockCacheRepository) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	args := m.Called(ctx, keys)
	return args.Get(0).(map[string]string), args.Error(1)
}

func (m *MockCacheRepository) SetMultiple(ctx context.Context, values map[string]string, expiration time.Duration) error {
	args := m.Called(ctx, values, expiration)
	return args.Error(0)
}

func (m *MockCacheRepository) FlushAll(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// Test setup helper
func setupURLService() (*URLService, *MockURLRepository, *MockAnalyticsRepository, *MockCacheRepository) {
	mockURLRepo := &MockURLRepository{}
	mockAnalyticsRepo := &MockAnalyticsRepository{}
	mockCacheRepo := &MockCacheRepository{}

	service := NewURLService(
		mockURLRepo,
		mockAnalyticsRepo,
		mockCacheRepo,
		8, // shortCodeLength
		"https://short.ly",
		5*time.Minute, // cacheTTL
	)

	return service, mockURLRepo, mockAnalyticsRepo, mockCacheRepo
}

func TestURLService_ShortenURL_Success(t *testing.T) {
	service, mockURLRepo, _, mockCacheRepo := setupURLService()
	ctx := context.Background()

	req := &models.CreateURLRequest{
		OriginalURL: "https://example.com",
		UserID:      stringPtr("user123"),
	}

	// Mock expectations
	mockURLRepo.On("GetByOriginalURL", ctx, req.OriginalURL, req.UserID).Return(nil, models.ErrURLNotFound)
	mockURLRepo.On("GetByShortCode", ctx, mock.AnythingOfType("string")).Return(nil, models.ErrURLNotFound)
	mockURLRepo.On("Create", ctx, mock.AnythingOfType("*models.URL")).Return(nil)
	mockCacheRepo.On("Set", ctx, mock.AnythingOfType("string"), req.OriginalURL, 5*time.Minute).Return(nil)

	// Execute
	url, shortURL, err := service.ShortenURL(ctx, req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, url)
	assert.NotEmpty(t, url.ShortCode)
	assert.Equal(t, req.OriginalURL, url.OriginalURL)
	assert.Equal(t, req.UserID, url.UserID)
	assert.Contains(t, shortURL, "https://short.ly/")
	assert.True(t, url.IsActive)

	mockURLRepo.AssertExpectations(t)
	mockCacheRepo.AssertExpectations(t)
}

func TestURLService_ShortenURL_CustomAlias(t *testing.T) {
	service, mockURLRepo, _, mockCacheRepo := setupURLService()
	ctx := context.Background()

	customAlias := "my-custom-link"
	req := &models.CreateURLRequest{
		OriginalURL: "https://example.com",
		CustomAlias: &customAlias,
		UserID:      stringPtr("user123"),
	}

	// Mock expectations
	mockURLRepo.On("GetByOriginalURL", ctx, req.OriginalURL, req.UserID).Return(nil, models.ErrURLNotFound)
	mockURLRepo.On("GetByShortCode", ctx, customAlias).Return(nil, models.ErrURLNotFound) // Check custom alias availability
	mockURLRepo.On("Create", ctx, mock.AnythingOfType("*models.URL")).Return(nil)
	mockCacheRepo.On("Set", ctx, mock.AnythingOfType("string"), req.OriginalURL, 5*time.Minute).Return(nil)

	// Execute
	url, shortURL, err := service.ShortenURL(ctx, req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, url)
	assert.Equal(t, customAlias, url.ShortCode)
	assert.Equal(t, &customAlias, url.CustomAlias)
	assert.Contains(t, shortURL, customAlias)

	mockURLRepo.AssertExpectations(t)
	mockCacheRepo.AssertExpectations(t)
}

func TestURLService_ShortenURL_InvalidURL(t *testing.T) {
	service, _, _, _ := setupURLService()
	ctx := context.Background()

	req := &models.CreateURLRequest{
		OriginalURL: "invalid-url",
		UserID:      stringPtr("user123"),
	}

	// Execute
	url, shortURL, err := service.ShortenURL(ctx, req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, url)
	assert.Empty(t, shortURL)
	assert.Contains(t, err.Error(), "VALIDATION_ERROR")
}

func TestURLService_GetOriginalURL_Success(t *testing.T) {
	service, mockURLRepo, mockAnalyticsRepo, mockCacheRepo := setupURLService()
	ctx := context.Background()

	shortCode := "abc123"
	originalURL := "https://example.com"
	
	mockURL := &models.URL{
		ID:          uuid.New(),
		ShortCode:   shortCode,
		OriginalURL: originalURL,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	clientInfo := &ClientInfo{
		IPAddress: stringPtr("192.168.1.1"),
		UserAgent: stringPtr("Mozilla/5.0"),
	}

	// Mock expectations
	mockCacheRepo.On("Get", ctx, "url:"+shortCode).Return("", assert.AnError) // Cache miss
	mockURLRepo.On("GetByShortCode", ctx, shortCode).Return(mockURL, nil)
	// These are called asynchronously, so we'll mock them but not assert on them
	mockURLRepo.On("IncrementClickCount", mock.Anything, shortCode).Return(nil).Maybe()
	mockURLRepo.On("UpdateLastAccessed", mock.Anything, shortCode).Return(nil).Maybe()
	mockAnalyticsRepo.On("RecordAccess", mock.Anything, mock.AnythingOfType("*models.Analytics")).Return(nil).Maybe()
	mockCacheRepo.On("Set", mock.Anything, "url:"+shortCode, originalURL, 5*time.Minute).Return(nil).Maybe()

	// Execute
	result, err := service.GetOriginalURL(ctx, shortCode, clientInfo)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, originalURL, result)

	mockURLRepo.AssertExpectations(t)
	mockCacheRepo.AssertExpectations(t)
}

func TestURLService_GetOriginalURL_FromCache(t *testing.T) {
	service, mockURLRepo, mockAnalyticsRepo, mockCacheRepo := setupURLService()
	ctx := context.Background()

	shortCode := "abc123"
	originalURL := "https://example.com"

	clientInfo := &ClientInfo{
		IPAddress: stringPtr("192.168.1.1"),
	}

	mockURL := &models.URL{
		ID:          uuid.New(),
		ShortCode:   shortCode,
		OriginalURL: originalURL,
		IsActive:    true,
		ExpiresAt:   nil, // Not expired
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Mock expectations - cache hit but we still check database for expiration/active status
	mockCacheRepo.On("Get", ctx, "url:"+shortCode).Return(originalURL, nil)
	mockURLRepo.On("GetByShortCode", ctx, shortCode).Return(mockURL, nil)
	mockURLRepo.On("IncrementClickCount", mock.Anything, shortCode).Return(nil)
	mockURLRepo.On("UpdateLastAccessed", mock.Anything, shortCode).Return(nil)
	mockAnalyticsRepo.On("RecordAccess", mock.Anything, mock.AnythingOfType("*models.Analytics")).Return(nil)

	// Execute
	result, err := service.GetOriginalURL(ctx, shortCode, clientInfo)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, originalURL, result)

	mockCacheRepo.AssertExpectations(t)
}

func TestURLService_GetOriginalURL_NotFound(t *testing.T) {
	service, mockURLRepo, _, mockCacheRepo := setupURLService()
	ctx := context.Background()

	shortCode := "notfound"

	// Mock expectations
	mockCacheRepo.On("Get", ctx, "url:"+shortCode).Return("", assert.AnError) // Cache miss
	mockURLRepo.On("GetByShortCode", ctx, shortCode).Return(nil, models.ErrURLNotFound)

	// Execute
	result, err := service.GetOriginalURL(ctx, shortCode, nil)

	// Assert
	assert.Error(t, err)
	assert.Empty(t, result)
	assert.Equal(t, models.ErrURLNotFound, err)

	mockURLRepo.AssertExpectations(t)
	mockCacheRepo.AssertExpectations(t)
}

func TestURLService_GetOriginalURL_Expired(t *testing.T) {
	service, mockURLRepo, _, mockCacheRepo := setupURLService()
	ctx := context.Background()

	shortCode := "expired"
	expiredTime := time.Now().Add(-1 * time.Hour)
	
	mockURL := &models.URL{
		ID:          uuid.New(),
		ShortCode:   shortCode,
		OriginalURL: "https://example.com",
		IsActive:    true,
		ExpiresAt:   &expiredTime,
		CreatedAt:   time.Now().Add(-2 * time.Hour),
		UpdatedAt:   time.Now().Add(-2 * time.Hour),
	}

	// Mock expectations
	mockCacheRepo.On("Get", ctx, "url:"+shortCode).Return("", assert.AnError) // Cache miss
	mockURLRepo.On("GetByShortCode", ctx, shortCode).Return(mockURL, nil)

	// Execute
	result, err := service.GetOriginalURL(ctx, shortCode, nil)

	// Assert
	assert.Error(t, err)
	assert.Empty(t, result)
	assert.Equal(t, models.ErrURLExpired, err)

	mockURLRepo.AssertExpectations(t)
	mockCacheRepo.AssertExpectations(t)
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
