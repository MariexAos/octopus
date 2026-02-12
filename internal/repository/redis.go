package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"octopus/internal/config"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

const (
	// Redis key prefixes
	ShortLinkKeyPrefix  = "sl:"
	ShortLinkCacheTTL   = 24 * time.Hour
	PVKeyPrefix         = "sl:pv:"
	UVKeyPrefix         = "sl:uv:"
	SourceKeyPrefix     = "sl:source:"
	StatsExpireDuration = 24 * time.Hour
)

// RedisRepository handles Redis operations
type RedisRepository struct {
	client *redis.Client
	cfg    *config.RedisConfig
}

// NewRedisRepository creates a new Redis repository
func NewRedisRepository(cfg *config.RedisConfig) *RedisRepository {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Error().Err(err).Msg("Failed to connect to Redis")
	} else {
		log.Info().Msg("Redis connected successfully")
	}

	return &RedisRepository{
		client: rdb,
		cfg:    cfg,
	}
}

// GetClient returns the Redis client
func (r *RedisRepository) GetClient() *redis.Client {
	return r.client
}

// SaveShortLink saves a short link to Redis
func (r *RedisRepository) SaveShortLink(ctx context.Context, shortCode, originalURL string, ttl time.Duration) error {
	key := r.shortLinkKey(shortCode)
	return r.client.Set(ctx, key, originalURL, ttl).Err()
}

// GetShortLink retrieves a short link from Redis
func (r *RedisRepository) GetShortLink(ctx context.Context, shortCode string) (string, error) {
	key := r.shortLinkKey(shortCode)
	return r.client.Get(ctx, key).Result()
}

// ExistsShortLink checks if a short link exists in Redis
func (r *RedisRepository) ExistsShortLink(ctx context.Context, shortCode string) (bool, error) {
	key := r.shortLinkKey(shortCode)
	result, err := r.client.Exists(ctx, key).Result()
	return result > 0, err
}

// IncrementPV increments the page view count for a short link
func (r *RedisRepository) IncrementPV(ctx context.Context, shortCode string) (int64, error) {
	key := r.pvKey(shortCode)
	count, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	// Set expiration if this is the first increment
	if count == 1 {
		r.client.Expire(ctx, key, StatsExpireDuration)
	}
	return count, nil
}

// GetPV gets the page view count for a short link
func (r *RedisRepository) GetPV(ctx context.Context, shortCode string) (int64, error) {
	key := r.pvKey(shortCode)
	return r.client.Get(ctx, key).Int64()
}

// AddUV adds a unique visitor for a short link
func (r *RedisRepository) AddUV(ctx context.Context, shortCode, visitorID string) (bool, error) {
	key := r.uvKey(shortCode)
	day := time.Now().Format("2006-01-02")
	dailyKey := fmt.Sprintf("%s:%s", key, day)

	added, err := r.client.SAdd(ctx, dailyKey, visitorID).Result()
	if err != nil {
		return false, err
	}
	// Set expiration
	r.client.Expire(ctx, dailyKey, StatsExpireDuration)

	return added > 0, nil
}

// GetUV gets the unique visitor count for a short link
func (r *RedisRepository) GetUV(ctx context.Context, shortCode string) (int64, error) {
	pattern := fmt.Sprintf("%s:*", r.uvKey(shortCode))
	var keys []string

	iter := r.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if err := iter.Err(); err != nil {
		return 0, err
	}

	var totalUV int64
	for _, key := range keys {
		count, err := r.client.SCard(ctx, key).Result()
		if err != nil {
			continue
		}
		totalUV += count
	}

	return totalUV, nil
}

// AddSource adds a source visit for a short link
func (r *RedisRepository) AddSource(ctx context.Context, shortCode, source string) error {
	key := r.sourceKey(shortCode)
	day := time.Now().Format("2006-01-02")
	dailyKey := fmt.Sprintf("%s:%s:%s", key, source, day)

	count, err := r.client.Incr(ctx, dailyKey).Result()
	if err != nil {
		return err
	}
	// Set expiration
	if count == 1 {
		r.client.Expire(ctx, dailyKey, StatsExpireDuration)
	}

	return nil
}

// GetSources gets the top sources for a short link
func (r *RedisRepository) GetSources(ctx context.Context, shortCode string) (map[string]int64, error) {
	pattern := fmt.Sprintf("%s:*", r.sourceKey(shortCode))
	sources := make(map[string]int64)

	iter := r.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		count, err := r.client.Get(ctx, key).Int64()
		if err != nil {
			continue
		}
		// Extract source name from key
		parts := r.sourceKey(shortCode)
		sourceName := key[len(parts)+1:]
		// Remove the date part
		if idx := strings.LastIndex(sourceName, ":"); idx > 0 {
			sourceName = sourceName[:idx]
		}
		sources[sourceName] += count
	}

	return sources, iter.Err()
}

// Close closes the Redis connection
func (r *RedisRepository) Close() error {
	return r.client.Close()
}

// Helper functions to build Redis keys

func (r *RedisRepository) shortLinkKey(shortCode string) string {
	return ShortLinkKeyPrefix + shortCode
}

func (r *RedisRepository) pvKey(shortCode string) string {
	return PVKeyPrefix + shortCode
}

func (r *RedisRepository) uvKey(shortCode string) string {
	return UVKeyPrefix + shortCode
}

func (r *RedisRepository) sourceKey(shortCode string) string {
	return SourceKeyPrefix + shortCode
}
