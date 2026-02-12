package repository

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"octopus/internal/config"
)

func newTestRedisRepo(t *testing.T) (*RedisRepository, *miniredis.Miniredis) {
	s := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: s.Addr()})

	return &RedisRepository{
		client: client,
		cfg: &config.RedisConfig{
			Addr:     s.Addr(),
			Password: "",
			DB:       0,
		},
	}, s
}

func TestNewRedisRepository(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	cfg := &config.RedisConfig{
		Addr:     s.Addr(),
		Password: "",
		DB:       0,
	}

	repo := NewRedisRepository(cfg)

	assert.NotNil(t, repo)
	assert.NotNil(t, repo.client)
	assert.Equal(t, cfg, repo.cfg)

	// Close connection after test
	repo.Close()
}

func TestRedisRepository_SaveShortLink(t *testing.T) {
	repo, _ := newTestRedisRepo(t)
	defer repo.Close()

	ctx := context.Background()

	err := repo.SaveShortLink(ctx, "ABCD", "https://example.com", ShortLinkCacheTTL)
	require.NoError(t, err)

	// Verify it was saved
	url, err := repo.GetShortLink(ctx, "ABCD")
	assert.NoError(t, err)
	assert.Equal(t, "https://example.com", url)
}

func TestRedisRepository_GetShortLink(t *testing.T) {
	repo, s := newTestRedisRepo(t)
	defer repo.Close()

	ctx := context.Background()

	t.Run("existing short link", func(t *testing.T) {
		s.Set(ShortLinkKeyPrefix+"ABCD", "https://example.com")

		url, err := repo.GetShortLink(ctx, "ABCD")
		assert.NoError(t, err)
		assert.Equal(t, "https://example.com", url)
	})

	t.Run("non-existent short link", func(t *testing.T) {
		_, err := repo.GetShortLink(ctx, "NONEXIST")
		assert.Error(t, err)
		assert.Equal(t, redis.Nil, err)
	})
}

func TestRedisRepository_ExistsShortLink(t *testing.T) {
	repo, s := newTestRedisRepo(t)
	defer repo.Close()

	ctx := context.Background()

	t.Run("existing short link", func(t *testing.T) {
		s.Set(ShortLinkKeyPrefix+"ABCD", "https://example.com")

		exists, err := repo.ExistsShortLink(ctx, "ABCD")
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("non-existent short link", func(t *testing.T) {
		exists, err := repo.ExistsShortLink(ctx, "NONEXIST")
		assert.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestRedisRepository_IncrementPV(t *testing.T) {
	repo, _ := newTestRedisRepo(t)
	defer repo.Close()

	ctx := context.Background()

	t.Run("first increment", func(t *testing.T) {
		count, err := repo.IncrementPV(ctx, "ABCD")
		assert.NoError(t, err)
		assert.Equal(t, int64(1), count)

		// Verify it was set
		pv, err := repo.GetPV(ctx, "ABCD")
		assert.NoError(t, err)
		assert.Equal(t, int64(1), pv)
	})

	t.Run("subsequent increments", func(t *testing.T) {
		_, _ = repo.IncrementPV(ctx, "XYZ")

		count, err := repo.IncrementPV(ctx, "XYZ")
		assert.NoError(t, err)
		assert.Equal(t, int64(2), count)

		pv, err := repo.GetPV(ctx, "XYZ")
		assert.NoError(t, err)
		assert.Equal(t, int64(2), pv)
	})
}

func TestRedisRepository_GetPV(t *testing.T) {
	repo, s := newTestRedisRepo(t)
	defer repo.Close()

	ctx := context.Background()

	t.Run("existing PV", func(t *testing.T) {
		s.Set(PVKeyPrefix+"ABCD", "100")

		pv, err := repo.GetPV(ctx, "ABCD")
		assert.NoError(t, err)
		assert.Equal(t, int64(100), pv)
	})

	t.Run("non-existent PV", func(t *testing.T) {
		_, err := repo.GetPV(ctx, "NONEXIST")
		assert.Error(t, err)
		assert.Equal(t, redis.Nil, err)
	})
}

func TestRedisRepository_AddUV(t *testing.T) {
	repo, _ := newTestRedisRepo(t)
	defer repo.Close()

	ctx := context.Background()

	t.Run("first visitor", func(t *testing.T) {
		added, err := repo.AddUV(ctx, "ABCD", "visitor1")
		assert.NoError(t, err)
		assert.True(t, added)

		// Verify it was added
		uv, err := repo.GetUV(ctx, "ABCD")
		assert.NoError(t, err)
		assert.Equal(t, int64(1), uv)
	})

	t.Run("same visitor again", func(t *testing.T) {
		_, _ = repo.AddUV(ctx, "XYZ", "visitor1")

		added, err := repo.AddUV(ctx, "XYZ", "visitor1")
		assert.NoError(t, err)
		assert.False(t, added)

		uv, err := repo.GetUV(ctx, "XYZ")
		assert.NoError(t, err)
		assert.Equal(t, int64(1), uv)
	})

	t.Run("different visitor", func(t *testing.T) {
		_, _ = repo.AddUV(ctx, "NEW", "visitor1")

		added, err := repo.AddUV(ctx, "NEW", "visitor2")
		assert.NoError(t, err)
		assert.True(t, added)

		uv, err := repo.GetUV(ctx, "NEW")
		assert.NoError(t, err)
		assert.Equal(t, int64(2), uv)
	})
}

func TestRedisRepository_GetUV(t *testing.T) {
	repo, _ := newTestRedisRepo(t)
	defer repo.Close()

	ctx := context.Background()

	t.Run("get UV with multiple visitors", func(t *testing.T) {
		_, _ = repo.AddUV(ctx, "ABCD", "visitor1")
		_, _ = repo.AddUV(ctx, "ABCD", "visitor2")
		_, _ = repo.AddUV(ctx, "ABCD", "visitor3")

		uv, err := repo.GetUV(ctx, "ABCD")
		assert.NoError(t, err)
		assert.Equal(t, int64(3), uv)
	})

	t.Run("get UV for non-existent", func(t *testing.T) {
		uv, err := repo.GetUV(ctx, "NONEXIST")
		assert.NoError(t, err)
		assert.Equal(t, int64(0), uv)
	})
}

func TestRedisRepository_AddSource(t *testing.T) {
	repo, _ := newTestRedisRepo(t)
	defer repo.Close()

	ctx := context.Background()

	t.Run("add source visit", func(t *testing.T) {
		err := repo.AddSource(ctx, "ABCD", "google")
		assert.NoError(t, err)

		sources, err := repo.GetSources(ctx, "ABCD")
		assert.NoError(t, err)
		assert.Equal(t, int64(1), sources["google"])
	})

	t.Run("add multiple sources", func(t *testing.T) {
		_ = repo.AddSource(ctx, "XYZ", "google")
		_ = repo.AddSource(ctx, "XYZ", "google")
		_ = repo.AddSource(ctx, "XYZ", "direct")

		sources, err := repo.GetSources(ctx, "XYZ")
		assert.NoError(t, err)
		assert.Equal(t, int64(2), sources["google"])
		assert.Equal(t, int64(1), sources["direct"])
	})
}

func TestRedisRepository_GetSources(t *testing.T) {
	repo, _ := newTestRedisRepo(t)
	defer repo.Close()

	ctx := context.Background()

	t.Run("get sources with data", func(t *testing.T) {
		_ = repo.AddSource(ctx, "ABCD", "google")
		_ = repo.AddSource(ctx, "ABCD", "google")
		_ = repo.AddSource(ctx, "ABCD", "baidu")
		_ = repo.AddSource(ctx, "ABCD", "direct")

		sources, err := repo.GetSources(ctx, "ABCD")
		assert.NoError(t, err)
		assert.Equal(t, int64(2), sources["google"])
		assert.Equal(t, int64(1), sources["baidu"])
		assert.Equal(t, int64(1), sources["direct"])
	})

	t.Run("get sources for non-existent", func(t *testing.T) {
		sources, err := repo.GetSources(ctx, "NONEXIST")
		assert.NoError(t, err)
		assert.Empty(t, sources)
	})
}

func TestRedisRepository_Close(t *testing.T) {
	repo, s := newTestRedisRepo(t)

	err := repo.Close()
	assert.NoError(t, err)

	// Verify connection is closed
	ctx := context.Background()
	_, err = repo.GetShortLink(ctx, "ABCD")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "closed")

	s.Close()
}

func TestRedisRepository_shortLinkKey(t *testing.T) {
	repo, _ := newTestRedisRepo(t)
	defer repo.Close()

	assert.Equal(t, "sl:ABCD", repo.shortLinkKey("ABCD"))
	assert.Equal(t, "sl:TEST", repo.shortLinkKey("TEST"))
}

func TestRedisRepository_pvKey(t *testing.T) {
	repo, _ := newTestRedisRepo(t)
	defer repo.Close()

	assert.Equal(t, "sl:pv:ABCD", repo.pvKey("ABCD"))
	assert.Equal(t, "sl:pv:TEST", repo.pvKey("TEST"))
}

func TestRedisRepository_uvKey(t *testing.T) {
	repo, _ := newTestRedisRepo(t)
	defer repo.Close()

	assert.Equal(t, "sl:uv:ABCD", repo.uvKey("ABCD"))
	assert.Equal(t, "sl:uv:TEST", repo.uvKey("TEST"))
}

func TestRedisRepository_sourceKey(t *testing.T) {
	repo, _ := newTestRedisRepo(t)
	defer repo.Close()

	assert.Equal(t, "sl:source:ABCD", repo.sourceKey("ABCD"))
	assert.Equal(t, "sl:source:TEST", repo.sourceKey("TEST"))
}

func TestRedisRepository_GetClient(t *testing.T) {
	repo, _ := newTestRedisRepo(t)
	defer repo.Close()

	client := repo.GetClient()
	assert.NotNil(t, client)

	ctx := context.Background()
	err := client.Ping(ctx).Err()
	assert.NoError(t, err)
}
