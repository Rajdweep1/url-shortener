package test

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/suite"

	"github.com/rajweepmondal/url-shortener/internal/config"
	"github.com/rajweepmondal/url-shortener/internal/models"
	"github.com/rajweepmondal/url-shortener/internal/repository/postgres"
	redisRepo "github.com/rajweepmondal/url-shortener/internal/repository/redis"
	"github.com/rajweepmondal/url-shortener/internal/service"
	"github.com/rajweepmondal/url-shortener/internal/utils"
)

// IntegrationTestSuite contains integration tests
type IntegrationTestSuite struct {
	suite.Suite
	db          *sql.DB
	redisClient *redis.Client
	urlService  *service.URLService
	cleanup     func()
}

// SetupSuite runs before all tests in the suite
func (suite *IntegrationTestSuite) SetupSuite() {
	// Skip integration tests if not explicitly enabled
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		suite.T().Skip("Integration tests skipped. Set RUN_INTEGRATION_TESTS=true to run.")
	}

	// Setup test configuration
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			URL:             getEnvOrDefault("TEST_DATABASE_URL", "postgres://postgres:password@localhost:5432/url_shortener_test?sslmode=disable"),
			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxLifetime: 5 * time.Minute,
			ConnMaxIdleTime: 1 * time.Minute,
		},
		Redis: config.RedisConfig{
			URL:          getEnvOrDefault("TEST_REDIS_URL", "redis://localhost:6379/1"),
			PoolSize:     10,
			MinIdleConn:  2,
			DialTimeout:  5 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
		},
		App: config.AppConfig{
			BaseURL:         "https://test.ly",
			ShortCodeLength: 8,
			CacheTTL:        5 * time.Minute,
			RateLimit:       100,
			RateWindow:      time.Minute,
		},
	}

	// Connect to databases
	dbConn, err := utils.NewDatabaseConnection(cfg)
	suite.Require().NoError(err, "Failed to connect to test databases")

	suite.db = dbConn.PostgreSQL
	suite.redisClient = dbConn.Redis

	// Setup cleanup function
	suite.cleanup = func() {
		dbConn.Close()
	}

	// Initialize repositories
	urlRepo := postgres.NewURLRepository(suite.db)
	analyticsRepo := postgres.NewAnalyticsRepository(suite.db)
	cacheRepo := redisRepo.NewCacheRepository(suite.redisClient)

	// Initialize service
	suite.urlService = service.NewURLService(
		urlRepo,
		analyticsRepo,
		cacheRepo,
		cfg.App.ShortCodeLength,
		cfg.App.BaseURL,
		cfg.App.CacheTTL,
	)

	// Clean up any existing test data
	suite.cleanupTestData()
}

// TearDownSuite runs after all tests in the suite
func (suite *IntegrationTestSuite) TearDownSuite() {
	if suite.cleanup != nil {
		suite.cleanup()
	}
}

// SetupTest runs before each test
func (suite *IntegrationTestSuite) SetupTest() {
	suite.cleanupTestData()
}

// TearDownTest runs after each test
func (suite *IntegrationTestSuite) TearDownTest() {
	suite.cleanupTestData()
}

// cleanupTestData removes all test data
func (suite *IntegrationTestSuite) cleanupTestData() {
	ctx := context.Background()

	// Clean PostgreSQL - delete all test data (be more aggressive for integration tests)
	_, err := suite.db.ExecContext(ctx, "DELETE FROM analytics")
	suite.Require().NoError(err)

	_, err = suite.db.ExecContext(ctx, "DELETE FROM urls")
	suite.Require().NoError(err)

	// Clean Redis
	err = suite.redisClient.FlushDB(ctx).Err()
	suite.Require().NoError(err)
}

// TestURLService_ShortenURL_Integration tests URL shortening end-to-end
func (suite *IntegrationTestSuite) TestURLService_ShortenURL_Integration() {
	ctx := context.Background()

	req := &models.CreateURLRequest{
		OriginalURL: "https://example.com/integration-test",
		UserID:      stringPtr("test-user-123"),
	}

	// Create shortened URL
	url, shortURL, err := suite.urlService.ShortenURL(ctx, req)
	suite.Require().NoError(err)
	suite.Assert().NotNil(url)
	suite.Assert().NotEmpty(shortURL)
	suite.Assert().Equal(req.OriginalURL, url.OriginalURL)
	suite.Assert().Equal(req.UserID, url.UserID)
	suite.Assert().True(url.IsActive)
	suite.Assert().Contains(shortURL, "https://test.ly/")

	// Verify URL was saved to database
	savedURL, err := suite.urlService.GetURLInfo(ctx, url.ShortCode, req.UserID)
	suite.Require().NoError(err)
	suite.Assert().Equal(url.ID, savedURL.ID)
	suite.Assert().Equal(url.ShortCode, savedURL.ShortCode)
	suite.Assert().Equal(url.OriginalURL, savedURL.OriginalURL)
}

// TestURLService_GetOriginalURL_Integration tests URL retrieval and caching
func (suite *IntegrationTestSuite) TestURLService_GetOriginalURL_Integration() {
	ctx := context.Background()

	// First create a URL
	req := &models.CreateURLRequest{
		OriginalURL: "https://example.com/get-test",
		UserID:      stringPtr("test-user-456"),
	}

	url, _, err := suite.urlService.ShortenURL(ctx, req)
	suite.Require().NoError(err)

	clientInfo := &service.ClientInfo{
		IPAddress: stringPtr("192.168.1.100"),
		UserAgent: stringPtr("Test-Agent/1.0"),
	}

	// Test retrieval (should hit database first time)
	originalURL, err := suite.urlService.GetOriginalURL(ctx, url.ShortCode, clientInfo)
	suite.Require().NoError(err)
	suite.Assert().Equal(req.OriginalURL, originalURL)

	// Test retrieval again (should hit cache this time)
	originalURL2, err := suite.urlService.GetOriginalURL(ctx, url.ShortCode, clientInfo)
	suite.Require().NoError(err)
	suite.Assert().Equal(req.OriginalURL, originalURL2)

	// Verify click count was incremented
	time.Sleep(500 * time.Millisecond) // Allow async operations to complete
	updatedURL, err := suite.urlService.GetURLInfo(ctx, url.ShortCode, req.UserID)
	suite.Require().NoError(err)
	suite.Assert().Greater(updatedURL.ClickCount, int64(0))
}

// TestURLService_CustomAlias_Integration tests custom alias functionality
func (suite *IntegrationTestSuite) TestURLService_CustomAlias_Integration() {
	ctx := context.Background()

	customAlias := "test-custom-alias"
	req := &models.CreateURLRequest{
		OriginalURL: "https://example.com/custom-alias-test",
		CustomAlias: &customAlias,
		UserID:      stringPtr("test-user-789"),
	}

	// Create URL with custom alias
	url, shortURL, err := suite.urlService.ShortenURL(ctx, req)
	suite.Require().NoError(err)
	suite.Assert().Equal(customAlias, url.ShortCode)
	suite.Assert().Equal(&customAlias, url.CustomAlias)
	suite.Assert().Contains(shortURL, customAlias)

	// Verify we can retrieve by custom alias
	originalURL, err := suite.urlService.GetOriginalURL(ctx, customAlias, nil)
	suite.Require().NoError(err)
	suite.Assert().Equal(req.OriginalURL, originalURL)
}

// TestURLService_ExpiredURL_Integration tests URL expiration
func (suite *IntegrationTestSuite) TestURLService_ExpiredURL_Integration() {
	ctx := context.Background()

	// Create URL that expires in 1 second
	expiresIn := 1 * time.Second
	req := &models.CreateURLRequest{
		OriginalURL: "https://example.com/expiry-test",
		ExpiresIn:   &expiresIn,
		UserID:      stringPtr("test-user-expiry"),
	}

	url, _, err := suite.urlService.ShortenURL(ctx, req)
	suite.Require().NoError(err)
	suite.Assert().NotNil(url.ExpiresAt)

	// Should work immediately
	originalURL, err := suite.urlService.GetOriginalURL(ctx, url.ShortCode, nil)
	suite.Require().NoError(err)
	suite.Assert().Equal(req.OriginalURL, originalURL)

	// Wait for expiration
	time.Sleep(2 * time.Second)

	// Should now be expired
	_, err = suite.urlService.GetOriginalURL(ctx, url.ShortCode, nil)
	suite.Require().Error(err)
	suite.Assert().Equal(models.ErrURLExpired, err)
}

// TestURLService_UpdateURL_Integration tests URL updates
func (suite *IntegrationTestSuite) TestURLService_UpdateURL_Integration() {
	ctx := context.Background()

	// Create initial URL
	req := &models.CreateURLRequest{
		OriginalURL: "https://example.com/update-test-original",
		UserID:      stringPtr("test-user-update"),
	}

	url, _, err := suite.urlService.ShortenURL(ctx, req)
	suite.Require().NoError(err)

	// Update the URL
	newOriginalURL := "https://example.com/update-test-modified"
	updates := &models.URL{
		OriginalURL: newOriginalURL,
		IsActive:    true,
	}

	updatedURL, err := suite.urlService.UpdateURL(ctx, url.ShortCode, updates, req.UserID)
	suite.Require().NoError(err)
	suite.Assert().Equal(newOriginalURL, updatedURL.OriginalURL)

	// Verify the update persisted
	retrievedURL, err := suite.urlService.GetOriginalURL(ctx, url.ShortCode, nil)
	suite.Require().NoError(err)
	suite.Assert().Equal(newOriginalURL, retrievedURL)
}

// TestURLService_DeleteURL_Integration tests URL deletion
func (suite *IntegrationTestSuite) TestURLService_DeleteURL_Integration() {
	ctx := context.Background()

	// Create URL
	req := &models.CreateURLRequest{
		OriginalURL: "https://example.com/delete-test",
		UserID:      stringPtr("test-user-delete"),
	}

	url, _, err := suite.urlService.ShortenURL(ctx, req)
	suite.Require().NoError(err)

	// Verify it exists
	_, err = suite.urlService.GetOriginalURL(ctx, url.ShortCode, nil)
	suite.Require().NoError(err)

	// Delete it
	err = suite.urlService.DeleteURL(ctx, url.ShortCode, req.UserID)
	suite.Require().NoError(err)

	// Verify it's gone
	_, err = suite.urlService.GetOriginalURL(ctx, url.ShortCode, nil)
	suite.Require().Error(err)
}

// Helper functions
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func stringPtr(s string) *string {
	return &s
}

// TestIntegrationSuite runs the integration test suite
func TestIntegrationSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
