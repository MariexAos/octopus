package service

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"octopus/internal/model"

	"github.com/rs/zerolog/log"
)

// AnalyticsService handles analytics operations
type AnalyticsService struct {
	redisRepo RedisRepositoryInterface
}

// NewAnalyticsService creates a new Analytics Service
func NewAnalyticsService(redisRepo RedisRepositoryInterface) *AnalyticsService {
	return &AnalyticsService{
		redisRepo: redisRepo,
	}
}

// RecordAccess records a single access event
func (as *AnalyticsService) RecordAccess(ctx context.Context, shortCode, clientIP, userAgent, referer string) error {
	// Increment PV
	if _, err := as.redisRepo.IncrementPV(ctx, shortCode); err != nil {
		log.Error().Err(err).Str("short_code", shortCode).Msg("Failed to increment PV")
	}

	// Add UV (using IP as visitor ID)
	visitorID := fmt.Sprintf("%s:%s", time.Now().Format("2006-01-02"), clientIP)
	if _, err := as.redisRepo.AddUV(ctx, shortCode, visitorID); err != nil {
		log.Error().Err(err).Str("short_code", shortCode).Msg("Failed to add UV")
	}

	// Add source
	source := as.extractSource(referer)
	if source != "" {
		if err := as.redisRepo.AddSource(ctx, shortCode, source); err != nil {
			log.Error().Err(err).Str("short_code", shortCode).Str("source", source).Msg("Failed to add source")
		}
	}

	return nil
}

// GetStats returns PV and UV statistics for a short code
func (as *AnalyticsService) GetStats(ctx context.Context, shortCode string) (*model.Stats, error) {
	pv, err := as.redisRepo.GetPV(ctx, shortCode)
	if err != nil {
		log.Error().Err(err).Str("short_code", shortCode).Msg("Failed to get PV")
		pv = 0
	}

	uv, err := as.redisRepo.GetUV(ctx, shortCode)
	if err != nil {
		log.Error().Err(err).Str("short_code", shortCode).Msg("Failed to get UV")
		uv = 0
	}

	return &model.Stats{PV: pv, UV: uv}, nil
}

// GetAnalytics returns detailed analytics for a short code
func (as *AnalyticsService) GetAnalytics(ctx context.Context, shortCode string) (*model.AnalyticsResponse, error) {
	stats, err := as.GetStats(ctx, shortCode)
	if err != nil {
		return nil, err
	}

	sources, err := as.redisRepo.GetSources(ctx, shortCode)
	if err != nil {
		log.Error().Err(err).Str("short_code", shortCode).Msg("Failed to get sources")
		sources = make(map[string]int64)
	}

	// Convert to top sources
	topSources := as.getTopSources(sources, 10)

	return &model.AnalyticsResponse{
		ShortCode:  shortCode,
		PV:         stats.PV,
		UV:         stats.UV,
		TopSources: topSources,
	}, nil
}

// extractSource extracts the source from referer URL
func (as *AnalyticsService) extractSource(referer string) string {
	if referer == "" {
		return "direct"
	}

	u, err := url.Parse(referer)
	if err != nil {
		return "unknown"
	}

	host := u.Host
	if strings.HasPrefix(host, "www.") {
		host = host[4:]
	}

	// Known sources
	switch {
	case strings.Contains(host, "google"):
		return "google"
	case strings.Contains(host, "baidu"):
		return "baidu"
	case strings.Contains(host, "bing"):
		return "bing"
	case strings.Contains(host, "weibo"):
		return "weibo"
	case strings.Contains(host, "weixin") || strings.Contains(host, "mp.weixin.qq.com"):
		return "wechat"
	case strings.Contains(host, "qq"):
		return "qq"
	case strings.Contains(host, "zhihu"):
		return "zhihu"
	default:
		// Return domain name for other sources
		parts := strings.Split(host, ".")
		if len(parts) >= 2 {
			return parts[len(parts)-2]
		}
		return host
	}
}

// getTopSources returns the top N sources
func (as *AnalyticsService) getTopSources(sources map[string]int64, limit int) []model.SourceStat {
	if len(sources) == 0 {
		return []model.SourceStat{}
	}

	// Convert to slice and sort
	stats := make([]model.SourceStat, 0, len(sources))
	for source, count := range sources {
		stats = append(stats, model.SourceStat{Source: source, Count: count})
	}

	// Simple sort by count descending
	for i := 0; i < len(stats); i++ {
		for j := i + 1; j < len(stats); j++ {
			if stats[j].Count > stats[i].Count {
				stats[i], stats[j] = stats[j], stats[i]
			}
		}
	}

	// Limit results
	if len(stats) > limit {
		stats = stats[:limit]
	}

	return stats
}
