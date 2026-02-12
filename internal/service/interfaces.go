package service

import (
	"context"
	"time"

	"octopus/internal/model"

	"github.com/redis/go-redis/v9"
)

// MySQLRepositoryInterface defines the interface for MySQL operations (for testing)
type MySQLRepositoryInterface interface {
	SaveShortLink(ctx context.Context, sl *model.ShortLink) error
	GetShortLinkByCode(ctx context.Context, shortCode string) (*model.ShortLink, error)
	GetShortLinkByURL(ctx context.Context, url string) (*model.ShortLink, error)
	CheckExistsByCode(ctx context.Context, shortCode string) (bool, error)
}

// RedisRepositoryInterface defines the interface for Redis operations (for testing)
type RedisRepositoryInterface interface {
	GetClient() *redis.Client
	SaveShortLink(ctx context.Context, shortCode, originalURL string, ttl time.Duration) error
	GetShortLink(ctx context.Context, shortCode string) (string, error)
	IncrementPV(ctx context.Context, shortCode string) (int64, error)
	GetPV(ctx context.Context, shortCode string) (int64, error)
	AddUV(ctx context.Context, shortCode, visitorID string) (bool, error)
	GetUV(ctx context.Context, shortCode string) (int64, error)
	AddSource(ctx context.Context, shortCode, source string) error
	GetSources(ctx context.Context, shortCode string) (map[string]int64, error)
}

// BloomServiceInterface defines the interface for Bloom Filter operations (for testing)
type BloomServiceInterface interface {
	Add(ctx context.Context, shortCode string) error
	Exists(ctx context.Context, shortCode string) (bool, error)
	GetCapacity() int64
	IsAvailable(ctx context.Context) bool
	Reset(ctx context.Context) error
}

// ShortLinkServiceInterface defines the interface for short link operations
type ShortLinkServiceInterface interface {
	Generate(ctx context.Context, req *model.GenerateRequest) (*model.GenerateResponse, error)
	Get(ctx context.Context, shortCode string) (*model.ShortLink, error)
	ExpandURL(ctx context.Context, shortCode string, queryParams map[string]string) (string, error)
}

// AnalyticsServiceInterface defines the interface for analytics operations
type AnalyticsServiceInterface interface {
	RecordAccess(ctx context.Context, shortCode, clientIP, userAgent, referer string) error
	GetStats(ctx context.Context, shortCode string) (*model.Stats, error)
	GetAnalytics(ctx context.Context, shortCode string) (*model.AnalyticsResponse, error)
}
