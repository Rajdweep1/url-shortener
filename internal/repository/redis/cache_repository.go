package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rajweepmondal/url-shortener/internal/repository/interfaces"
)

// CacheRepository implements the CacheRepository interface for Redis
type CacheRepository struct {
	client *redis.Client
}

// NewCacheRepository creates a new Redis cache repository
func NewCacheRepository(client *redis.Client) interfaces.CacheRepository {
	return &CacheRepository{client: client}
}

// Get retrieves a value from cache
func (r *CacheRepository) Get(ctx context.Context, key string) (string, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", fmt.Errorf("key not found: %s", key)
		}
		return "", fmt.Errorf("failed to get key %s: %w", key, err)
	}
	return val, nil
}

// Set stores a value in cache with expiration
func (r *CacheRepository) Set(ctx context.Context, key string, value string, expiration time.Duration) error {
	err := r.client.Set(ctx, key, value, expiration).Err()
	if err != nil {
		return fmt.Errorf("failed to set key %s: %w", key, err)
	}
	return nil
}

// Delete removes a value from cache
func (r *CacheRepository) Delete(ctx context.Context, key string) error {
	err := r.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete key %s: %w", key, err)
	}
	return nil
}

// Exists checks if a key exists in cache
func (r *CacheRepository) Exists(ctx context.Context, key string) (bool, error) {
	count, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check existence of key %s: %w", key, err)
	}
	return count > 0, nil
}

// Increment increments a counter in cache
func (r *CacheRepository) Increment(ctx context.Context, key string) (int64, error) {
	val, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to increment key %s: %w", key, err)
	}
	return val, nil
}

// IncrementWithExpiry increments a counter with expiration
func (r *CacheRepository) IncrementWithExpiry(ctx context.Context, key string, expiration time.Duration) (int64, error) {
	pipe := r.client.Pipeline()
	incrCmd := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, expiration)
	
	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to increment key %s with expiry: %w", key, err)
	}
	
	return incrCmd.Val(), nil
}

// GetMultiple retrieves multiple values from cache
func (r *CacheRepository) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	if len(keys) == 0 {
		return make(map[string]string), nil
	}
	
	pipe := r.client.Pipeline()
	cmds := make(map[string]*redis.StringCmd)
	
	for _, key := range keys {
		cmds[key] = pipe.Get(ctx, key)
	}
	
	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to get multiple keys: %w", err)
	}
	
	result := make(map[string]string)
	for key, cmd := range cmds {
		val, err := cmd.Result()
		if err == nil {
			result[key] = val
		}
		// Ignore redis.Nil errors for individual keys
	}
	
	return result, nil
}

// SetMultiple stores multiple values in cache
func (r *CacheRepository) SetMultiple(ctx context.Context, values map[string]string, expiration time.Duration) error {
	if len(values) == 0 {
		return nil
	}
	
	pipe := r.client.Pipeline()
	
	for key, value := range values {
		pipe.Set(ctx, key, value, expiration)
	}
	
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to set multiple keys: %w", err)
	}
	
	return nil
}

// FlushAll clears all cache entries
func (r *CacheRepository) FlushAll(ctx context.Context) error {
	err := r.client.FlushAll(ctx).Err()
	if err != nil {
		return fmt.Errorf("failed to flush all keys: %w", err)
	}
	return nil
}

// Additional utility methods for URL shortener specific caching

// GetURL retrieves a cached URL by short code
func (r *CacheRepository) GetURL(ctx context.Context, shortCode string) (string, error) {
	key := fmt.Sprintf("url:%s", shortCode)
	return r.Get(ctx, key)
}

// SetURL caches a URL by short code
func (r *CacheRepository) SetURL(ctx context.Context, shortCode, originalURL string, expiration time.Duration) error {
	key := fmt.Sprintf("url:%s", shortCode)
	return r.Set(ctx, key, originalURL, expiration)
}

// DeleteURL removes a cached URL
func (r *CacheRepository) DeleteURL(ctx context.Context, shortCode string) error {
	key := fmt.Sprintf("url:%s", shortCode)
	return r.Delete(ctx, key)
}

// GetURLStats retrieves cached URL statistics
func (r *CacheRepository) GetURLStats(ctx context.Context, shortCode string) (map[string]string, error) {
	keys := []string{
		fmt.Sprintf("stats:%s:total", shortCode),
		fmt.Sprintf("stats:%s:unique", shortCode),
		fmt.Sprintf("stats:%s:today", shortCode),
		fmt.Sprintf("stats:%s:week", shortCode),
	}
	return r.GetMultiple(ctx, keys)
}

// SetURLStats caches URL statistics
func (r *CacheRepository) SetURLStats(ctx context.Context, shortCode string, stats map[string]string, expiration time.Duration) error {
	cacheData := make(map[string]string)
	for key, value := range stats {
		cacheKey := fmt.Sprintf("stats:%s:%s", shortCode, key)
		cacheData[cacheKey] = value
	}
	return r.SetMultiple(ctx, cacheData, expiration)
}

// IncrementClickCount increments the click count for a URL in cache
func (r *CacheRepository) IncrementClickCount(ctx context.Context, shortCode string, expiration time.Duration) (int64, error) {
	key := fmt.Sprintf("clicks:%s", shortCode)
	return r.IncrementWithExpiry(ctx, key, expiration)
}

// GetClickCount retrieves the cached click count for a URL
func (r *CacheRepository) GetClickCount(ctx context.Context, shortCode string) (int64, error) {
	key := fmt.Sprintf("clicks:%s", shortCode)
	val, err := r.Get(ctx, key)
	if err != nil {
		return 0, err
	}
	
	// Convert string to int64
	var count int64
	_, err = fmt.Sscanf(val, "%d", &count)
	if err != nil {
		return 0, fmt.Errorf("failed to parse click count: %w", err)
	}
	
	return count, nil
}
