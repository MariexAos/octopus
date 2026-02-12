package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"net/url"
	"time"

	"octopus/internal/encoder"
	"octopus/internal/model"
	"octopus/internal/repository"

	"github.com/rs/zerolog/log"
)

// hashString computes FNV-1a hash of a string
func hashString(s string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	return h.Sum64()
}

var (
	// ErrInvalidURL is returned when the URL is invalid
	ErrInvalidURL = errors.New("invalid URL")
	// ErrShortLinkNotFound is returned when the short link is not found
	ErrShortLinkNotFound = errors.New("short link not found")
	// ErrShortLinkExpired is returned when the short link has expired
	ErrShortLinkExpired = errors.New("short link has expired")
	// ErrMaxCapacityReached is returned when maximum capacity is reached
	ErrMaxCapacityReached = errors.New("maximum capacity reached")
)

// ShortLinkService handles short link operations
type ShortLinkService struct {
	encoder   *encoder.Base32Encoder
	mysqlRepo MySQLRepositoryInterface
	redisRepo RedisRepositoryInterface
	bloomSvc  BloomServiceInterface
	domain    string
}

// NewShortLinkService creates a new ShortLink Service
func NewShortLinkService(
	mysqlRepo MySQLRepositoryInterface,
	redisRepo RedisRepositoryInterface,
	bloomSvc BloomServiceInterface,
	domain string,
) *ShortLinkService {
	return &ShortLinkService{
		encoder:   encoder.NewBase32Encoder(),
		mysqlRepo: mysqlRepo,
		redisRepo: redisRepo,
		bloomSvc:  bloomSvc,
		domain:    domain,
	}
}

// Generate generates a short link for the given URL
func (s *ShortLinkService) Generate(ctx context.Context, req *model.GenerateRequest) (*model.GenerateResponse, error) {
	// Validate URL
	if req.URL == "" {
		return nil, ErrInvalidURL
	}

	// Parse expire time if provided
	var expireAt *time.Time
	if req.ExpireAt != "" {
		t, err := time.Parse(time.RFC3339, req.ExpireAt)
		if err != nil {
			return nil, fmt.Errorf("invalid expire_at format: %w", err)
		}
		expireAt = &t
	}

	// Build cache key for URL + params
	cacheKey := s.buildCacheKey(req.URL, req.Params)

	// Check cache first
	if cachedCode, err := s.redisRepo.GetShortLink(ctx, cacheKey); err == nil && cachedCode != "" {
		// Found in cache, return existing short link
		if sl, err := s.mysqlRepo.GetShortLinkByCode(ctx, cachedCode); err == nil {
			return s.buildResponse(sl), nil
		}
	}

	// Check if URL already exists
	if existing, err := s.mysqlRepo.GetShortLinkByURL(ctx, req.URL); err == nil {
		// Cache it
		s.redisRepo.SaveShortLink(ctx, cacheKey, existing.ShortCode, repository.ShortLinkCacheTTL)
		return s.buildResponse(existing), nil
	}

	// Generate new short code with collision handling
	shortCode, err := s.generateWithCollision(ctx, req.URL)
	if err != nil {
		return nil, err
	}

	// Prepare params JSON
	var paramsJSON []byte
	if req.Params != nil {
		paramsJSON, _ = json.Marshal(req.Params)
	}

	// Create short link entity
	now := time.Now()
	sl := &model.ShortLink{
		ShortCode:   shortCode,
		OriginalURL: req.URL,
		Params:      paramsJSON,
		CreatedAt:   now,
		ExpireAt:    expireAt,
		Status:      1,
	}

	// Save to MySQL
	if err := s.mysqlRepo.SaveShortLink(ctx, sl); err != nil {
		log.Error().Err(err).Str("short_code", shortCode).Msg("Failed to save short link to MySQL")
		return nil, fmt.Errorf("failed to save short link: %w", err)
	}

	// Save to Redis cache
	s.redisRepo.SaveShortLink(ctx, cacheKey, shortCode, repository.ShortLinkCacheTTL)
	s.redisRepo.SaveShortLink(ctx, shortCode, req.URL, repository.ShortLinkCacheTTL)

	// Add to Bloom Filter
	if err := s.bloomSvc.Add(ctx, shortCode); err != nil {
		log.Warn().Err(err).Str("short_code", shortCode).Msg("Failed to add to Bloom Filter")
	}

	return s.buildResponse(sl), nil
}

// Get retrieves the original URL for a short code
func (s *ShortLinkService) Get(ctx context.Context, shortCode string) (*model.ShortLink, error) {
	// Try cache first
	if url, err := s.redisRepo.GetShortLink(ctx, shortCode); err == nil && url != "" {
		// Reconstruct short link
		sl := &model.ShortLink{
			ShortCode:   shortCode,
			OriginalURL: url,
		}
		return sl, nil
	}

	// Try MySQL
	sl, err := s.mysqlRepo.GetShortLinkByCode(ctx, shortCode)
	if err != nil {
		return nil, ErrShortLinkNotFound
	}

	// Check if expired
	if !sl.IsActive() {
		return nil, ErrShortLinkExpired
	}

	// Cache it
	s.redisRepo.SaveShortLink(ctx, shortCode, sl.OriginalURL, repository.ShortLinkCacheTTL)

	return sl, nil
}

// ExpandURL expands a short URL with query parameters
func (s *ShortLinkService) ExpandURL(ctx context.Context, shortCode string, queryParams map[string]string) (string, error) {
	sl, err := s.Get(ctx, shortCode)
	if err != nil {
		return "", err
	}

	targetURL := sl.OriginalURL

	// Parse existing URL
	u, err := url.Parse(targetURL)
	if err != nil {
		return targetURL, nil // Return as-is if parse fails
	}

	// Build query string
	query := u.Query()
	for key, value := range queryParams {
		query.Set(key, value)
	}
	u.RawQuery = query.Encode()

	return u.String(), nil
}

// generateWithCollision generates a short code with collision handling
func (s *ShortLinkService) generateWithCollision(ctx context.Context, url string) (string, error) {
	// Start with 4 characters
	for length := encoder.MinLength; length <= encoder.MaxLength; length++ {
		hash := hashString(url)

		for i := 0; i < 1000; i++ { // Retry up to 1000 times per length
			shortCode := s.encoder.Encode(hash+uint64(i), length)

			// Check Bloom Filter first (fast check)
			exists, err := s.bloomSvc.Exists(ctx, shortCode)
			if err != nil || !exists {
				// Bloom Filter says not exists, check DB to be sure
				actualExists, _ := s.mysqlRepo.CheckExistsByCode(ctx, shortCode)
				if !actualExists {
					return shortCode, nil
				}
			}

			// Collision detected, increment hash
		}

		// All codes of this length are used, try longer length
	}

	return "", ErrMaxCapacityReached
}

// buildCacheKey builds a cache key for URL and params
func (s *ShortLinkService) buildCacheKey(url string, params map[string]interface{}) string {
	if len(params) == 0 {
		return url
	}
	return fmt.Sprintf("%s:%v", url, params)
}

// buildResponse builds a generate response from a short link entity
func (s *ShortLinkService) buildResponse(sl *model.ShortLink) *model.GenerateResponse {
	shortLink := fmt.Sprintf("%s/%s", s.domain, sl.ShortCode)

	resp := &model.GenerateResponse{
		ShortLink:   shortLink,
		ShortCode:   sl.ShortCode,
		OriginalURL: sl.OriginalURL,
	}

	if sl.ExpireAt != nil {
		resp.ExpireAt = *sl.ExpireAt
	}

	return resp
}
