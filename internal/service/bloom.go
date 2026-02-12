package service

import (
	"context"
	"fmt"
	"time"

	"octopus/internal/config"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// BloomService handles Bloom Filter operations
type BloomService struct {
	client    RedisClient
	capacity  int64
	errorRate float64
}

// RedisClient defines the interface for Redis client operations
type RedisClient interface {
	Do(ctx context.Context, args ...interface{}) *redis.Cmd
	Exists(ctx context.Context, keys ...string) *redis.IntCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
}

// NewBloomService creates a new Bloom Service
func NewBloomService(client RedisClient, cfg *config.BloomConfig) *BloomService {
	bs := &BloomService{
		client:    client,
		capacity:  cfg.Capacity,
		errorRate: cfg.ErrorRate,
	}

	// Initialize Bloom Filter if needed
	bs.initBloomFilter(context.Background())

	return bs
}

const bloomFilterKey = "shortlink:bloom"

// initBloomFilter initializes the Bloom Filter
func (bs *BloomService) initBloomFilter(ctx context.Context) {
	// Check if Bloom Filter exists
	exists, err := bs.client.Exists(ctx, bloomFilterKey).Result()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to check Bloom Filter existence")
		return
	}

	if exists > 0 {
		log.Info().Msg("Bloom Filter already exists")
		return
	}

	// Create Bloom Filter
	cmd := bs.client.Do(ctx, "BF.RESERVE", bloomFilterKey, bs.errorRate, bs.capacity)
	if err := cmd.Err(); err != nil {
		// BF.RESERVE may not be available, use BF.ADD instead
		log.Warn().Err(err).Msg("BF.RESERVE not available, using dynamic Bloom Filter")
	} else {
		log.Info().Msgf("Bloom Filter created with capacity=%d, error_rate=%f", bs.capacity, bs.errorRate)
	}
}

// Add adds a short code to the Bloom Filter
func (bs *BloomService) Add(ctx context.Context, shortCode string) error {
	// Try BF.ADD first (RedisBloom module)
	cmd := bs.client.Do(ctx, "BF.ADD", bloomFilterKey, shortCode)
	if err := cmd.Err(); err != nil {
		// Fallback to regular SET if Bloom Filter not available
		log.Warn().Err(err).Msg("BF.ADD not available, using SET as fallback")
		key := bs.fallbackKey(shortCode)
		return bs.client.Set(ctx, key, 1, 0).Err()
	}
	return nil
}

// Exists checks if a short code might exist in the Bloom Filter
func (bs *BloomService) Exists(ctx context.Context, shortCode string) (bool, error) {
	// Try BF.EXISTS first
	cmd := bs.client.Do(ctx, "BF.EXISTS", bloomFilterKey, shortCode)
	result, err := cmd.Int()
	if err == nil {
		return result == 1, nil
	}

	// Fallback to regular GET if Bloom Filter not available
	log.Warn().Err(err).Msg("BF.EXISTS not available, using GET as fallback")
	key := bs.fallbackKey(shortCode)
	exists, err := bs.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

// Fallback key when Bloom Filter is not available
func (bs *BloomService) fallbackKey(shortCode string) string {
	return fmt.Sprintf("shortlink:bloom:fb:%s", shortCode)
}

// GetCapacity returns the capacity of the Bloom Filter
func (bs *BloomService) GetCapacity() int64 {
	return bs.capacity
}

// IsAvailable checks if Bloom Filter is available
func (bs *BloomService) IsAvailable(ctx context.Context) bool {
	cmd := bs.client.Do(ctx, "BF.INFO", bloomFilterKey)
	if cmd.Err() != nil {
		return false
	}
	return true
}

// Reset resets the Bloom Filter (use with caution)
func (bs *BloomService) Reset(ctx context.Context) error {
	return bs.client.Del(ctx, bloomFilterKey).Err()
}
