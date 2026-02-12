package repository

import (
	"context"
	"time"

	"octopus/internal/model"
)

// MySQLRepositoryInterface defines the interface for MySQL operations
type MySQLRepositoryInterface interface {
	GetDB() interface{}
	SaveShortLink(ctx context.Context, sl *model.ShortLink) error
	GetShortLinkByCode(ctx context.Context, shortCode string) (*model.ShortLink, error)
	GetShortLinkByURL(ctx context.Context, url string) (*model.ShortLink, error)
	CheckExistsByCode(ctx context.Context, shortCode string) (bool, error)
	SaveAccessLog(ctx context.Context, accessLog *model.AccessLog) error
	GetAccessLogs(ctx context.Context, shortCode string, limit int) ([]model.AccessLog, error)
	GetTotalLinksCount(ctx context.Context) (int64, error)
	CleanupExpiredLinks(ctx context.Context) (int64, error)
	Close() error
}

// RedisRepositoryInterface defines the interface for Redis operations
type RedisRepositoryInterface interface {
	GetClient() interface{}
	SaveShortLink(ctx context.Context, shortCode, originalURL string, ttl time.Duration) error
	GetShortLink(ctx context.Context, shortCode string) (string, error)
	ExistsShortLink(ctx context.Context, shortCode string) (bool, error)
	IncrementPV(ctx context.Context, shortCode string) (int64, error)
	GetPV(ctx context.Context, shortCode string) (int64, error)
	AddUV(ctx context.Context, shortCode, visitorID string) (bool, error)
	GetUV(ctx context.Context, shortCode string) (int64, error)
	AddSource(ctx context.Context, shortCode, source string) error
	GetSources(ctx context.Context, shortCode string) (map[string]int64, error)
	Close() error
}
