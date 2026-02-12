package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"octopus/internal/model"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"octopus/internal/mocks"
)

func TestNewShortLinkService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMySQL := mocks.NewMockMySQLRepositoryInterface(ctrl)
	mockRedis := mocks.NewMockRedisRepositoryInterface(ctrl)
	mockBloom := mocks.NewMockBloomServiceInterface(ctrl)

	svc := NewShortLinkService(mockMySQL, mockRedis, mockBloom, "https://s.example.com")

	assert.NotNil(t, svc)
	assert.Equal(t, mockMySQL, svc.mysqlRepo)
	assert.Equal(t, mockRedis, svc.redisRepo)
	assert.Equal(t, mockBloom, svc.bloomSvc)
	assert.Equal(t, "https://s.example.com", svc.domain)
}

func TestShortLinkService_Generate(t *testing.T) {
	tests := []struct {
		name      string
		req       *model.GenerateRequest
		setupMock func(*gomock.Controller) (MySQLRepositoryInterface, RedisRepositoryInterface, BloomServiceInterface)
		wantErr   error
		wantCode string
	}{
		{
			name: "empty URL",
			req:  &model.GenerateRequest{URL: ""},
			setupMock: func(ctrl *gomock.Controller) (MySQLRepositoryInterface, RedisRepositoryInterface, BloomServiceInterface) {
				return mocks.NewMockMySQLRepositoryInterface(ctrl),
					mocks.NewMockRedisRepositoryInterface(ctrl),
					mocks.NewMockBloomServiceInterface(ctrl)
			},
			wantErr: ErrInvalidURL,
		},
		{
			name: "invalid expire_at format",
			req:  &model.GenerateRequest{URL: "https://example.com", ExpireAt: "invalid"},
			setupMock: func(ctrl *gomock.Controller) (MySQLRepositoryInterface, RedisRepositoryInterface, BloomServiceInterface) {
				return mocks.NewMockMySQLRepositoryInterface(ctrl),
					mocks.NewMockRedisRepositoryInterface(ctrl),
					mocks.NewMockBloomServiceInterface(ctrl)
			},
			wantErr: errors.New("invalid expire_at format"),
		},
		{
			name: "cache hit",
			req:  &model.GenerateRequest{URL: "https://example.com"},
			setupMock: func(ctrl *gomock.Controller) (MySQLRepositoryInterface, RedisRepositoryInterface, BloomServiceInterface) {
				mockRedis := mocks.NewMockRedisRepositoryInterface(ctrl)
				mockMySQL := mocks.NewMockMySQLRepositoryInterface(ctrl)
				mockBloom := mocks.NewMockBloomServiceInterface(ctrl)

				mockRedis.EXPECT().GetShortLink(gomock.Any(), "https://example.com").Return("ABCD", nil)
				mockMySQL.EXPECT().GetShortLinkByCode(gomock.Any(), "ABCD").Return(&model.ShortLink{
					ShortCode:   "ABCD",
					OriginalURL: "https://example.com",
					Status:      1,
				}, nil)

				return mockMySQL, mockRedis, mockBloom
			},
			wantCode: "ABCD",
		},
		{
			name: "URL already exists in MySQL",
			req:  &model.GenerateRequest{URL: "https://example.com"},
			setupMock: func(ctrl *gomock.Controller) (MySQLRepositoryInterface, RedisRepositoryInterface, BloomServiceInterface) {
				mockRedis := mocks.NewMockRedisRepositoryInterface(ctrl)
				mockMySQL := mocks.NewMockMySQLRepositoryInterface(ctrl)
				mockBloom := mocks.NewMockBloomServiceInterface(ctrl)

				mockRedis.EXPECT().GetShortLink(gomock.Any(), "https://example.com").Return("", errors.New("not found"))
				mockMySQL.EXPECT().GetShortLinkByURL(gomock.Any(), "https://example.com").Return(&model.ShortLink{
					ID:          1,
					ShortCode:   "ABCD",
					OriginalURL: "https://example.com",
					Status:      1,
				}, nil)
				mockRedis.EXPECT().SaveShortLink(gomock.Any(), "https://example.com", "ABCD", gomock.Any()).Return(nil)

				return mockMySQL, mockRedis, mockBloom
			},
			wantCode: "ABCD",
		},
		{
			name: "generate new short link",
			req:  &model.GenerateRequest{URL: "https://example.com"},
			setupMock: func(ctrl *gomock.Controller) (MySQLRepositoryInterface, RedisRepositoryInterface, BloomServiceInterface) {
				mockRedis := mocks.NewMockRedisRepositoryInterface(ctrl)
				mockMySQL := mocks.NewMockMySQLRepositoryInterface(ctrl)
				mockBloom := mocks.NewMockBloomServiceInterface(ctrl)

				mockRedis.EXPECT().GetShortLink(gomock.Any(), "https://example.com").Return("", errors.New("not found"))
				mockMySQL.EXPECT().GetShortLinkByURL(gomock.Any(), "https://example.com").Return(nil, errors.New("not found"))
				mockBloom.EXPECT().Exists(gomock.Any(), gomock.Any()).Return(false, nil)
				mockMySQL.EXPECT().CheckExistsByCode(gomock.Any(), gomock.Any()).Return(false, nil)
				mockMySQL.EXPECT().SaveShortLink(gomock.Any(), gomock.Any()).Return(nil)
				// First SaveShortLink: cacheKey as key, shortCode as value
				mockRedis.EXPECT().SaveShortLink(gomock.Any(), gomock.Eq("https://example.com"), gomock.Any(), gomock.Any()).Return(nil)
				// Second SaveShortLink: shortCode as key, URL as value
				mockRedis.EXPECT().SaveShortLink(gomock.Any(), gomock.Any(), gomock.Eq("https://example.com"), gomock.Any()).Return(nil)
				mockBloom.EXPECT().Add(gomock.Any(), gomock.Any()).Return(nil)

				return mockMySQL, mockRedis, mockBloom
			},
			wantCode: "", // Will be set based on actual hash
		},
		{
			name: "generate with valid expire_at",
			req:  &model.GenerateRequest{URL: "https://example.com", ExpireAt: "2025-12-31T23:59:59Z"},
			setupMock: func(ctrl *gomock.Controller) (MySQLRepositoryInterface, RedisRepositoryInterface, BloomServiceInterface) {
				mockRedis := mocks.NewMockRedisRepositoryInterface(ctrl)
				mockMySQL := mocks.NewMockMySQLRepositoryInterface(ctrl)
				mockBloom := mocks.NewMockBloomServiceInterface(ctrl)

				mockRedis.EXPECT().GetShortLink(gomock.Any(), "https://example.com").Return("", errors.New("not found"))
				mockMySQL.EXPECT().GetShortLinkByURL(gomock.Any(), "https://example.com").Return(nil, errors.New("not found"))
				mockBloom.EXPECT().Exists(gomock.Any(), gomock.Any()).Return(false, nil)
				mockMySQL.EXPECT().CheckExistsByCode(gomock.Any(), gomock.Any()).Return(false, nil)
				mockMySQL.EXPECT().SaveShortLink(gomock.Any(), gomock.Any()).Return(nil)
				// First SaveShortLink: cacheKey as key, shortCode as value
				mockRedis.EXPECT().SaveShortLink(gomock.Any(), gomock.Eq("https://example.com"), gomock.Any(), gomock.Any()).Return(nil)
				// Second SaveShortLink: shortCode as key, URL as value
				mockRedis.EXPECT().SaveShortLink(gomock.Any(), gomock.Any(), gomock.Eq("https://example.com"), gomock.Any()).Return(nil)
				mockBloom.EXPECT().Add(gomock.Any(), gomock.Any()).Return(nil)

				return mockMySQL, mockRedis, mockBloom
			},
			wantCode: "",
		},
		{
			name: "generate with params",
			req: &model.GenerateRequest{
				URL:    "https://example.com",
				Params: map[string]interface{}{"utm_source": "google"},
			},
			setupMock: func(ctrl *gomock.Controller) (MySQLRepositoryInterface, RedisRepositoryInterface, BloomServiceInterface) {
				mockRedis := mocks.NewMockRedisRepositoryInterface(ctrl)
				mockMySQL := mocks.NewMockMySQLRepositoryInterface(ctrl)
				mockBloom := mocks.NewMockBloomServiceInterface(ctrl)

				mockRedis.EXPECT().GetShortLink(gomock.Any(), "https://example.com:map[utm_source:google]").Return("", errors.New("not found"))
				mockMySQL.EXPECT().GetShortLinkByURL(gomock.Any(), "https://example.com").Return(nil, errors.New("not found"))
				mockBloom.EXPECT().Exists(gomock.Any(), gomock.Any()).Return(false, nil)
				mockMySQL.EXPECT().CheckExistsByCode(gomock.Any(), gomock.Any()).Return(false, nil)
				mockMySQL.EXPECT().SaveShortLink(gomock.Any(), gomock.Any()).Return(nil)
				mockRedis.EXPECT().SaveShortLink(gomock.Any(), "https://example.com:map[utm_source:google]", gomock.Any(), gomock.Any()).Return(nil)
				mockRedis.EXPECT().SaveShortLink(gomock.Any(), gomock.Any(), "https://example.com", gomock.Any()).Return(nil)
				mockBloom.EXPECT().Add(gomock.Any(), gomock.Any()).Return(nil)

				return mockMySQL, mockRedis, mockBloom
			},
			wantCode: "",
		},
		{
			name: "max capacity reached",
			req:  &model.GenerateRequest{URL: "https://example.com"},
			setupMock: func(ctrl *gomock.Controller) (MySQLRepositoryInterface, RedisRepositoryInterface, BloomServiceInterface) {
				mockRedis := mocks.NewMockRedisRepositoryInterface(ctrl)
				mockMySQL := mocks.NewMockMySQLRepositoryInterface(ctrl)
				mockBloom := mocks.NewMockBloomServiceInterface(ctrl)

				mockRedis.EXPECT().GetShortLink(gomock.Any(), "https://example.com").Return("", errors.New("not found"))
				mockMySQL.EXPECT().GetShortLinkByURL(gomock.Any(), "https://example.com").Return(nil, errors.New("not found"))
				// Bloom filter says exists for all codes
				mockBloom.EXPECT().Exists(gomock.Any(), gomock.Any()).Return(true, nil).AnyTimes()
				mockMySQL.EXPECT().CheckExistsByCode(gomock.Any(), gomock.Any()).Return(true, nil).AnyTimes()

				return mockMySQL, mockRedis, mockBloom
			},
			wantErr: ErrMaxCapacityReached,
		},
		{
			name: "save to MySQL fails",
			req:  &model.GenerateRequest{URL: "https://example.com"},
			setupMock: func(ctrl *gomock.Controller) (MySQLRepositoryInterface, RedisRepositoryInterface, BloomServiceInterface) {
				mockRedis := mocks.NewMockRedisRepositoryInterface(ctrl)
				mockMySQL := mocks.NewMockMySQLRepositoryInterface(ctrl)
				mockBloom := mocks.NewMockBloomServiceInterface(ctrl)

				mockRedis.EXPECT().GetShortLink(gomock.Any(), "https://example.com").Return("", errors.New("not found"))
				mockMySQL.EXPECT().GetShortLinkByURL(gomock.Any(), "https://example.com").Return(nil, errors.New("not found"))
				mockBloom.EXPECT().Exists(gomock.Any(), gomock.Any()).Return(false, nil)
				mockMySQL.EXPECT().CheckExistsByCode(gomock.Any(), gomock.Any()).Return(false, nil)
				mockMySQL.EXPECT().SaveShortLink(gomock.Any(), gomock.Any()).Return(errors.New("db error"))

				return mockMySQL, mockRedis, mockBloom
			},
			wantErr: errors.New("failed to save short link"),
		},
		{
			name: "Bloom filter error - fallthrough to DB check",
			req:  &model.GenerateRequest{URL: "https://example.com"},
			setupMock: func(ctrl *gomock.Controller) (MySQLRepositoryInterface, RedisRepositoryInterface, BloomServiceInterface) {
				mockRedis := mocks.NewMockRedisRepositoryInterface(ctrl)
				mockMySQL := mocks.NewMockMySQLRepositoryInterface(ctrl)
				mockBloom := mocks.NewMockBloomServiceInterface(ctrl)

				mockRedis.EXPECT().GetShortLink(gomock.Any(), "https://example.com").Return("", errors.New("not found"))
				mockMySQL.EXPECT().GetShortLinkByURL(gomock.Any(), "https://example.com").Return(nil, errors.New("not found"))
				mockBloom.EXPECT().Exists(gomock.Any(), gomock.Any()).Return(false, errors.New("bloom error"))
				mockMySQL.EXPECT().CheckExistsByCode(gomock.Any(), gomock.Any()).Return(false, nil)
				mockMySQL.EXPECT().SaveShortLink(gomock.Any(), gomock.Any()).Return(nil)
				// First SaveShortLink: cacheKey as key, shortCode as value
				mockRedis.EXPECT().SaveShortLink(gomock.Any(), gomock.Eq("https://example.com"), gomock.Any(), gomock.Any()).Return(nil)
				// Second SaveShortLink: shortCode as key, URL as value
				mockRedis.EXPECT().SaveShortLink(gomock.Any(), gomock.Any(), gomock.Eq("https://example.com"), gomock.Any()).Return(nil)
				// Add to Bloom Filter (will be called even if Exists failed)
				mockBloom.EXPECT().Add(gomock.Any(), gomock.Any()).Return(nil)

				return mockMySQL, mockRedis, mockBloom
			},
			wantCode: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockMySQL, mockRedis, mockBloom := tt.setupMock(ctrl)
			svc := NewShortLinkService(mockMySQL, mockRedis, mockBloom, "https://s.example.com")

			resp, err := svc.Generate(context.Background(), tt.req)

			if tt.wantErr != nil {
				assert.Error(t, err)
				if tt.wantErr != ErrInvalidURL && tt.wantErr != ErrMaxCapacityReached {
					assert.Contains(t, err.Error(), tt.wantErr.Error())
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				if tt.wantCode != "" {
					assert.Equal(t, tt.wantCode, resp.ShortCode)
				} else {
					// Just verify short code is generated
					assert.NotEmpty(t, resp.ShortCode)
					assert.Equal(t, "https://example.com", resp.OriginalURL)
				}
			}
		})
	}
}

func TestShortLinkService_Get(t *testing.T) {
	tests := []struct {
		name      string
		shortCode string
		setupMock func(*gomock.Controller) (MySQLRepositoryInterface, RedisRepositoryInterface)
		wantErr   error
		wantURL   string
	}{
		{
			name:      "cache hit",
			shortCode: "ABCD",
			setupMock: func(ctrl *gomock.Controller) (MySQLRepositoryInterface, RedisRepositoryInterface) {
				mockRedis := mocks.NewMockRedisRepositoryInterface(ctrl)
				mockMySQL := mocks.NewMockMySQLRepositoryInterface(ctrl)

				mockRedis.EXPECT().GetShortLink(gomock.Any(), "ABCD").Return("https://example.com", nil)

				return mockMySQL, mockRedis
			},
			wantURL: "https://example.com",
		},
		{
			name:      "cache miss, MySQL hit",
			shortCode: "ABCD",
			setupMock: func(ctrl *gomock.Controller) (MySQLRepositoryInterface, RedisRepositoryInterface) {
				mockRedis := mocks.NewMockRedisRepositoryInterface(ctrl)
				mockMySQL := mocks.NewMockMySQLRepositoryInterface(ctrl)

				mockRedis.EXPECT().GetShortLink(gomock.Any(), "ABCD").Return("", errors.New("not found"))
				mockMySQL.EXPECT().GetShortLinkByCode(gomock.Any(), "ABCD").Return(&model.ShortLink{
					ShortCode:   "ABCD",
					OriginalURL: "https://example.com",
					Status:      1,
				}, nil)
				mockRedis.EXPECT().SaveShortLink(gomock.Any(), "ABCD", "https://example.com", gomock.Any()).Return(nil)

				return mockMySQL, mockRedis
			},
			wantURL: "https://example.com",
		},
		{
			name:      "short link not found",
			shortCode: "ABCD",
			setupMock: func(ctrl *gomock.Controller) (MySQLRepositoryInterface, RedisRepositoryInterface) {
				mockRedis := mocks.NewMockRedisRepositoryInterface(ctrl)
				mockMySQL := mocks.NewMockMySQLRepositoryInterface(ctrl)

				mockRedis.EXPECT().GetShortLink(gomock.Any(), "ABCD").Return("", errors.New("not found"))
				mockMySQL.EXPECT().GetShortLinkByCode(gomock.Any(), "ABCD").Return(nil, errors.New("not found"))

				return mockMySQL, mockRedis
			},
			wantErr: ErrShortLinkNotFound,
		},
		{
			name:      "short link expired",
			shortCode: "ABCD",
			setupMock: func(ctrl *gomock.Controller) (MySQLRepositoryInterface, RedisRepositoryInterface) {
				mockRedis := mocks.NewMockRedisRepositoryInterface(ctrl)
				mockMySQL := mocks.NewMockMySQLRepositoryInterface(ctrl)

				past := time.Now().Add(-1 * time.Hour)
				mockRedis.EXPECT().GetShortLink(gomock.Any(), "ABCD").Return("", errors.New("not found"))
				mockMySQL.EXPECT().GetShortLinkByCode(gomock.Any(), "ABCD").Return(&model.ShortLink{
					ShortCode:   "ABCD",
					OriginalURL: "https://example.com",
					Status:      1,
					ExpireAt:    &past,
				}, nil)

				return mockMySQL, mockRedis
			},
			wantErr: ErrShortLinkExpired,
		},
		{
			name:      "short link inactive",
			shortCode: "ABCD",
			setupMock: func(ctrl *gomock.Controller) (MySQLRepositoryInterface, RedisRepositoryInterface) {
				mockRedis := mocks.NewMockRedisRepositoryInterface(ctrl)
				mockMySQL := mocks.NewMockMySQLRepositoryInterface(ctrl)

				mockRedis.EXPECT().GetShortLink(gomock.Any(), "ABCD").Return("", errors.New("not found"))
				mockMySQL.EXPECT().GetShortLinkByCode(gomock.Any(), "ABCD").Return(&model.ShortLink{
					ShortCode:   "ABCD",
					OriginalURL: "https://example.com",
					Status:      0,
				}, nil)

				return mockMySQL, mockRedis
			},
			wantErr: ErrShortLinkExpired,
		},
		{
			name:      "cache and populate cache after MySQL hit",
			shortCode: "ABCD",
			setupMock: func(ctrl *gomock.Controller) (MySQLRepositoryInterface, RedisRepositoryInterface) {
				mockRedis := mocks.NewMockRedisRepositoryInterface(ctrl)
				mockMySQL := mocks.NewMockMySQLRepositoryInterface(ctrl)

				mockRedis.EXPECT().GetShortLink(gomock.Any(), "ABCD").Return("", errors.New("not found"))
				mockMySQL.EXPECT().GetShortLinkByCode(gomock.Any(), "ABCD").Return(&model.ShortLink{
					ShortCode:   "ABCD",
					OriginalURL: "https://example.com",
					Status:      1,
					ExpireAt:    nil,
				}, nil)
				mockRedis.EXPECT().SaveShortLink(gomock.Any(), "ABCD", "https://example.com", gomock.Any()).Return(nil)

				return mockMySQL, mockRedis
			},
			wantURL: "https://example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockMySQL, mockRedis := tt.setupMock(ctrl)
			mockBloom := mocks.NewMockBloomServiceInterface(ctrl)

			svc := NewShortLinkService(mockMySQL, mockRedis, mockBloom, "https://s.example.com")

			sl, err := svc.Get(context.Background(), tt.shortCode)

			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.IsType(t, tt.wantErr, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, sl)
				assert.Equal(t, tt.wantURL, sl.OriginalURL)
			}
		})
	}
}

func TestShortLinkService_ExpandURL(t *testing.T) {
	tests := []struct {
		name        string
		shortCode   string
		queryParams map[string]string
		setupMock   func(*gomock.Controller) (RedisRepositoryInterface)
		wantURL     string
		wantErr     error
	}{
		{
			name:        "expand with query params",
			shortCode:   "ABCD",
			queryParams: map[string]string{"utm_source": "google", "utm_campaign": "promo"},
			setupMock: func(ctrl *gomock.Controller) (RedisRepositoryInterface) {
				mockRedis := mocks.NewMockRedisRepositoryInterface(ctrl)

				mockRedis.EXPECT().GetShortLink(gomock.Any(), "ABCD").Return("https://example.com/page", nil)

				return mockRedis
			},
			wantURL: "https://example.com/page?utm_campaign=promo&utm_source=google",
		},
		{
			name:        "expand without query params",
			shortCode:   "ABCD",
			queryParams: map[string]string{},
			setupMock: func(ctrl *gomock.Controller) (RedisRepositoryInterface) {
				mockRedis := mocks.NewMockRedisRepositoryInterface(ctrl)

				mockRedis.EXPECT().GetShortLink(gomock.Any(), "ABCD").Return("https://example.com/page", nil)

				return mockRedis
			},
			wantURL: "https://example.com/page",
		},
		{
			name:        "expand with existing query params",
			shortCode:   "ABCD",
			queryParams: map[string]string{"new_param": "value"},
			setupMock: func(ctrl *gomock.Controller) (RedisRepositoryInterface) {
				mockRedis := mocks.NewMockRedisRepositoryInterface(ctrl)

				mockRedis.EXPECT().GetShortLink(gomock.Any(), "ABCD").Return("https://example.com/page?existing=value", nil)

				return mockRedis
			},
			wantURL: "https://example.com/page?existing=value&new_param=value",
		},
		{
			name:        "expand with empty query param value",
			shortCode:   "ABCD",
			queryParams: map[string]string{"empty": ""},
			setupMock: func(ctrl *gomock.Controller) (RedisRepositoryInterface) {
				mockRedis := mocks.NewMockRedisRepositoryInterface(ctrl)

				mockRedis.EXPECT().GetShortLink(gomock.Any(), "ABCD").Return("https://example.com/page", nil)

				return mockRedis
			},
			wantURL: "https://example.com/page?empty=",
		},
		{
			name:        "expand with special chars",
			shortCode:   "ABCD",
			queryParams: map[string]string{"query": "hello world"},
			setupMock: func(ctrl *gomock.Controller) (RedisRepositoryInterface) {
				mockRedis := mocks.NewMockRedisRepositoryInterface(ctrl)

				mockRedis.EXPECT().GetShortLink(gomock.Any(), "ABCD").Return("https://example.com", nil)

				return mockRedis
			},
			wantURL: "https://example.com?query=hello+world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRedis := tt.setupMock(ctrl)
			svc := NewShortLinkService(mocks.NewMockMySQLRepositoryInterface(ctrl), mockRedis, mocks.NewMockBloomServiceInterface(ctrl), "https://s.example.com")

			url, err := svc.ExpandURL(context.Background(), tt.shortCode, tt.queryParams)

			if tt.wantErr != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantURL, url)
			}
		})
	}
}

func TestShortLinkService_buildCacheKey(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMySQL := mocks.NewMockMySQLRepositoryInterface(ctrl)
	mockRedis := mocks.NewMockRedisRepositoryInterface(ctrl)
	mockBloom := mocks.NewMockBloomServiceInterface(ctrl)

	svc := NewShortLinkService(mockMySQL, mockRedis, mockBloom, "https://s.example.com")

	tests := []struct {
		name   string
		url    string
		params map[string]interface{}
		want   string
	}{
		{
			name:   "without params",
			url:    "https://example.com",
			params: nil,
			want:   "https://example.com",
		},
		{
			name:   "with empty params",
			url:    "https://example.com",
			params: map[string]interface{}{},
			want:   "https://example.com",
		},
		{
			name:   "with params",
			url:    "https://example.com",
			params: map[string]interface{}{"utm_source": "google"},
			want:   "https://example.com:map[utm_source:google]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.buildCacheKey(tt.url, tt.params)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestShortLinkService_buildResponse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMySQL := mocks.NewMockMySQLRepositoryInterface(ctrl)
	mockRedis := mocks.NewMockRedisRepositoryInterface(ctrl)
	mockBloom := mocks.NewMockBloomServiceInterface(ctrl)

	svc := NewShortLinkService(mockMySQL, mockRedis, mockBloom, "https://s.example.com")

	now := time.Now()

	tests := []struct {
		name string
		sl   *model.ShortLink
		want *model.GenerateResponse
	}{
		{
			name: "basic response",
			sl: &model.ShortLink{
				ShortCode:   "ABCD",
				OriginalURL: "https://example.com",
				Status:      1,
			},
			want: &model.GenerateResponse{
				ShortLink:   "https://s.example.com/ABCD",
				ShortCode:   "ABCD",
				OriginalURL: "https://example.com",
			},
		},
		{
			name: "response with expire_at",
			sl: &model.ShortLink{
				ShortCode:   "ABCD",
				OriginalURL: "https://example.com",
				Status:      1,
				ExpireAt:    &now,
			},
			want: &model.GenerateResponse{
				ShortLink:   "https://s.example.com/ABCD",
				ShortCode:   "ABCD",
				OriginalURL: "https://example.com",
				ExpireAt:    now,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.buildResponse(tt.sl)
			assert.Equal(t, tt.want.ShortLink, result.ShortLink)
			assert.Equal(t, tt.want.ShortCode, result.ShortCode)
			assert.Equal(t, tt.want.OriginalURL, result.OriginalURL)
			if tt.want.ExpireAt.IsZero() {
				assert.True(t, result.ExpireAt.IsZero())
			} else {
				assert.Equal(t, tt.want.ExpireAt, result.ExpireAt)
			}
		})
	}
}
