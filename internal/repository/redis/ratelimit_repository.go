package redis

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rajweepmondal/url-shortener/internal/repository/interfaces"
)

// RateLimitRepository implements the RateLimitRepository interface for Redis
type RateLimitRepository struct {
	client *redis.Client
}

// NewRateLimitRepository creates a new Redis rate limit repository
func NewRateLimitRepository(client *redis.Client) interfaces.RateLimitRepository {
	return &RateLimitRepository{client: client}
}

// CheckRateLimit checks if a request is within rate limits using sliding window
func (r *RateLimitRepository) CheckRateLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, int, error) {
	rateLimitKey := fmt.Sprintf("rate_limit:%s", key)
	
	// Use sliding window algorithm with Redis sorted sets
	now := time.Now().UnixNano()
	windowStart := now - window.Nanoseconds()
	
	pipe := r.client.Pipeline()
	
	// Remove expired entries
	pipe.ZRemRangeByScore(ctx, rateLimitKey, "0", strconv.FormatInt(windowStart, 10))
	
	// Count current requests in window
	countCmd := pipe.ZCard(ctx, rateLimitKey)
	
	// Add current request
	pipe.ZAdd(ctx, rateLimitKey, redis.Z{
		Score:  float64(now),
		Member: fmt.Sprintf("%d", now),
	})
	
	// Set expiration for the key
	pipe.Expire(ctx, rateLimitKey, window)
	
	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, 0, fmt.Errorf("failed to check rate limit: %w", err)
	}
	
	currentCount := int(countCmd.Val())
	
	// Check if within limit (we check before adding, so current count + 1)
	if currentCount >= limit {
		// Remove the request we just added since it exceeds the limit
		r.client.ZRem(ctx, rateLimitKey, fmt.Sprintf("%d", now))
		return false, currentCount, nil
	}
	
	return true, currentCount + 1, nil
}

// IncrementRateLimit increments the rate limit counter using token bucket algorithm
func (r *RateLimitRepository) IncrementRateLimit(ctx context.Context, key string, window time.Duration) (int, error) {
	rateLimitKey := fmt.Sprintf("rate_limit_counter:%s", key)
	
	// Use simple counter with expiration
	pipe := r.client.Pipeline()
	incrCmd := pipe.Incr(ctx, rateLimitKey)
	pipe.Expire(ctx, rateLimitKey, window)
	
	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to increment rate limit: %w", err)
	}
	
	return int(incrCmd.Val()), nil
}

// GetRateLimitInfo gets current rate limit information
func (r *RateLimitRepository) GetRateLimitInfo(ctx context.Context, key string) (int, time.Duration, error) {
	rateLimitKey := fmt.Sprintf("rate_limit_counter:%s", key)
	
	pipe := r.client.Pipeline()
	getCmd := pipe.Get(ctx, rateLimitKey)
	ttlCmd := pipe.TTL(ctx, rateLimitKey)
	
	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return 0, 0, fmt.Errorf("failed to get rate limit info: %w", err)
	}
	
	var count int
	if err != redis.Nil {
		countStr := getCmd.Val()
		count, err = strconv.Atoi(countStr)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to parse rate limit count: %w", err)
		}
	}
	
	ttl := ttlCmd.Val()
	if ttl < 0 {
		ttl = 0
	}
	
	return count, ttl, nil
}

// ResetRateLimit resets the rate limit for a key
func (r *RateLimitRepository) ResetRateLimit(ctx context.Context, key string) error {
	rateLimitKey := fmt.Sprintf("rate_limit_counter:%s", key)
	slidingWindowKey := fmt.Sprintf("rate_limit:%s", key)
	
	pipe := r.client.Pipeline()
	pipe.Del(ctx, rateLimitKey)
	pipe.Del(ctx, slidingWindowKey)
	
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to reset rate limit: %w", err)
	}
	
	return nil
}

// Additional utility methods for different rate limiting strategies

// CheckTokenBucket implements token bucket rate limiting
func (r *RateLimitRepository) CheckTokenBucket(ctx context.Context, key string, capacity, refillRate int, window time.Duration) (bool, error) {
	bucketKey := fmt.Sprintf("token_bucket:%s", key)
	
	// Lua script for atomic token bucket operations
	luaScript := `
		local key = KEYS[1]
		local capacity = tonumber(ARGV[1])
		local refill_rate = tonumber(ARGV[2])
		local window_seconds = tonumber(ARGV[3])
		local now = tonumber(ARGV[4])
		
		local bucket = redis.call('HMGET', key, 'tokens', 'last_refill')
		local tokens = tonumber(bucket[1]) or capacity
		local last_refill = tonumber(bucket[2]) or now
		
		-- Calculate tokens to add based on time elapsed
		local elapsed = now - last_refill
		local tokens_to_add = math.floor(elapsed * refill_rate / window_seconds)
		tokens = math.min(capacity, tokens + tokens_to_add)
		
		-- Check if we can consume a token
		if tokens > 0 then
			tokens = tokens - 1
			redis.call('HMSET', key, 'tokens', tokens, 'last_refill', now)
			redis.call('EXPIRE', key, window_seconds * 2)
			return 1
		else
			redis.call('HMSET', key, 'tokens', tokens, 'last_refill', now)
			redis.call('EXPIRE', key, window_seconds * 2)
			return 0
		end
	`
	
	result, err := r.client.Eval(ctx, luaScript, []string{bucketKey}, 
		capacity, refillRate, int(window.Seconds()), time.Now().Unix()).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check token bucket: %w", err)
	}
	
	return result.(int64) == 1, nil
}

// GetTokenBucketInfo gets current token bucket information
func (r *RateLimitRepository) GetTokenBucketInfo(ctx context.Context, key string) (int, time.Time, error) {
	bucketKey := fmt.Sprintf("token_bucket:%s", key)
	
	result, err := r.client.HMGet(ctx, bucketKey, "tokens", "last_refill").Result()
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("failed to get token bucket info: %w", err)
	}
	
	var tokens int
	var lastRefill time.Time
	
	if result[0] != nil {
		tokens, err = strconv.Atoi(result[0].(string))
		if err != nil {
			return 0, time.Time{}, fmt.Errorf("failed to parse tokens: %w", err)
		}
	}
	
	if result[1] != nil {
		timestamp, err := strconv.ParseInt(result[1].(string), 10, 64)
		if err != nil {
			return 0, time.Time{}, fmt.Errorf("failed to parse last refill: %w", err)
		}
		lastRefill = time.Unix(timestamp, 0)
	}
	
	return tokens, lastRefill, nil
}

// CheckFixedWindow implements fixed window rate limiting
func (r *RateLimitRepository) CheckFixedWindow(ctx context.Context, key string, limit int, window time.Duration) (bool, int, error) {
	// Create window-based key
	windowStart := time.Now().Truncate(window).Unix()
	windowKey := fmt.Sprintf("fixed_window:%s:%d", key, windowStart)
	
	pipe := r.client.Pipeline()
	incrCmd := pipe.Incr(ctx, windowKey)
	pipe.Expire(ctx, windowKey, window)
	
	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, 0, fmt.Errorf("failed to check fixed window: %w", err)
	}
	
	currentCount := int(incrCmd.Val())
	
	if currentCount > limit {
		return false, currentCount, nil
	}
	
	return true, currentCount, nil
}
