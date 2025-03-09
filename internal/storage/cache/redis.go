package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nghiack7/game-ad-service/internal/domain/models"
	"github.com/nghiack7/game-ad-service/internal/storage"
	"github.com/redis/go-redis/v9"
)

// RedisRepository implements the CacheRepository interface with Redis
type RedisRepository struct {
	client *redis.Client
	ttl    time.Duration
}

// RedisOptions contains options for Redis configuration
type RedisOptions struct {
	// Address is the Redis server address
	Address string

	// Password is the Redis password
	Password string

	// DB is the Redis database number
	DB int

	// DefaultTTL is the default expiration time for cache entries
	DefaultTTL time.Duration
}

// DefaultRedisOptions returns sensible defaults for Redis
func DefaultRedisOptions() RedisOptions {
	return RedisOptions{
		Address:    "localhost:6379",
		Password:   "",
		DB:         0,
		DefaultTTL: 24 * time.Hour,
	}
}

// NewRedisRepository creates a new Redis repository
func NewRedisRepository(options RedisOptions) (*RedisRepository, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     options.Address,
		Password: options.Password,
		DB:       options.DB,
	})

	// Test connection
	_, err := client.Ping(context.Background()).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisRepository{
		client: client,
		ttl:    options.DefaultTTL,
	}, nil
}

// Close closes the Redis connection
func (r *RedisRepository) Close() error {
	return r.client.Close()
}

// Set stores an ad in the cache with the given TTL
func (r *RedisRepository) Set(ctx context.Context, adID string, ad *models.Ad, ttlSeconds time.Duration) error {
	// Serialize the ad to JSON
	data, err := json.Marshal(ad)
	if err != nil {
		return fmt.Errorf("failed to serialize ad: %w", err)
	}

	// Set TTL based on the parameter or use default
	ttl := r.ttl
	if ttlSeconds > 0 {
		ttl = ttlSeconds
	}

	// Store in Redis
	key := fmt.Sprintf("ad:%s", adID)
	err = r.client.Set(ctx, key, data, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to store ad in Redis: %w", err)
	}

	return nil
}

// Get retrieves an ad from the cache
func (r *RedisRepository) Get(ctx context.Context, adID string) (*models.Ad, error) {
	key := fmt.Sprintf("ad:%s", adID)
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, storage.ErrAdNotFound
		}
		return nil, fmt.Errorf("failed to retrieve ad from Redis: %w", err)
	}

	// Deserialize the ad from JSON
	var ad models.Ad
	err = json.Unmarshal(data, &ad)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize ad: %w", err)
	}

	return &ad, nil
}

// Delete removes an ad from the cache
func (r *RedisRepository) Delete(ctx context.Context, adID string) error {
	key := fmt.Sprintf("ad:%s", adID)
	err := r.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete ad from Redis: %w", err)
	}

	return nil
}

// PushToQueue adds an ad ID to a priority queue
func (r *RedisRepository) PushToQueue(ctx context.Context, queueName string, adID string, priority int) error {
	// Use Redis sorted set for priority queue
	// Lower priority values = higher priority (earlier processing)
	score := float64(priority)

	err := r.client.ZAdd(ctx, queueName, redis.Z{
		Score:  score,
		Member: adID,
	}).Err()

	if err != nil {
		return fmt.Errorf("failed to push to Redis queue: %w", err)
	}

	return nil
}

// PopFromQueue retrieves and removes the highest priority item from a queue
func (r *RedisRepository) PopFromQueue(ctx context.Context, queueName string) (string, error) {
	// Use ZPOPMIN to get the lowest score (highest priority) item
	results, err := r.client.ZPopMin(ctx, queueName, 1).Result()
	if err != nil {
		return "", fmt.Errorf("failed to pop from Redis queue: %w", err)
	}

	if len(results) == 0 {
		return "", storage.ErrAdNotFound
	}

	// The Member is the ad ID
	adID, ok := results[0].Member.(string)
	if !ok {
		return "", fmt.Errorf("invalid ad ID in Redis queue")
	}

	return adID, nil
}

// IncrementCounter increments a counter and returns the new value
func (r *RedisRepository) IncrementCounter(ctx context.Context, key string) (int64, error) {
	// Use Redis INCR command
	val, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to increment Redis counter: %w", err)
	}

	return val, nil
}

// GetGauge gets the value of a gauge
func (r *RedisRepository) GetGauge(ctx context.Context, key string) (int64, error) {
	// Use Redis GET command
	val, err := r.client.Get(ctx, key).Int64()
	if err != nil {
		if err == redis.Nil {
			// Key doesn't exist, default to 0
			return 0, nil
		}
		return 0, fmt.Errorf("failed to get Redis gauge: %w", err)
	}

	return val, nil
}

// SetGauge sets the value of a gauge
func (r *RedisRepository) SetGauge(ctx context.Context, key string, value int64) error {
	// Use Redis SET command
	err := r.client.Set(ctx, key, value, 0).Err()
	if err != nil {
		return fmt.Errorf("failed to set Redis gauge: %w", err)
	}

	return nil
}
