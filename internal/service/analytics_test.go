package service

import (
	"context"
	"errors"
	"testing"

	"octopus/internal/model"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"octopus/internal/mocks"
)

func TestNewAnalyticsService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRedisRepositoryInterface(ctrl)
	svc := NewAnalyticsService(mockRepo)

	assert.NotNil(t, svc)
	assert.Equal(t, mockRepo, svc.redisRepo)
}

func TestAnalyticsService_RecordAccess(t *testing.T) {
	tests := []struct {
		name      string
		shortCode string
		clientIP  string
		userAgent string
		referer   string
		setupMock func(*gomock.Controller) *mocks.MockRedisRepositoryInterface
		expectErr bool
	}{
		{
			name:      "successful record access",
			shortCode: "ABCD",
			clientIP:  "192.168.1.1",
			userAgent: "Mozilla/5.0",
			referer:   "https://google.com",
			setupMock: func(ctrl *gomock.Controller) *mocks.MockRedisRepositoryInterface {
				mockRepo := mocks.NewMockRedisRepositoryInterface(ctrl)
				mockRepo.EXPECT().IncrementPV(gomock.Any(), "ABCD").Return(int64(1), nil)
				mockRepo.EXPECT().AddUV(gomock.Any(), "ABCD", gomock.Any()).Return(true, nil)
				mockRepo.EXPECT().AddSource(gomock.Any(), "ABCD", "google").Return(nil)
				return mockRepo
			},
			expectErr: false,
		},
		{
			name:      "record access with direct referer",
			shortCode: "ABCD",
			clientIP:  "192.168.1.1",
			userAgent: "Mozilla/5.0",
			referer:   "",
			setupMock: func(ctrl *gomock.Controller) *mocks.MockRedisRepositoryInterface {
				mockRepo := mocks.NewMockRedisRepositoryInterface(ctrl)
				mockRepo.EXPECT().IncrementPV(gomock.Any(), "ABCD").Return(int64(1), nil)
				mockRepo.EXPECT().AddUV(gomock.Any(), "ABCD", gomock.Any()).Return(true, nil)
				mockRepo.EXPECT().AddSource(gomock.Any(), "ABCD", "direct").Return(nil)
				return mockRepo
			},
			expectErr: false,
		},
		{
			name:      "record access with invalid referer",
			shortCode: "ABCD",
			clientIP:  "192.168.1.1",
			userAgent: "Mozilla/5.0",
			referer:   "://invalid",
			setupMock: func(ctrl *gomock.Controller) *mocks.MockRedisRepositoryInterface {
				mockRepo := mocks.NewMockRedisRepositoryInterface(ctrl)
				mockRepo.EXPECT().IncrementPV(gomock.Any(), "ABCD").Return(int64(1), nil)
				mockRepo.EXPECT().AddUV(gomock.Any(), "ABCD", gomock.Any()).Return(true, nil)
				mockRepo.EXPECT().AddSource(gomock.Any(), "ABCD", "unknown").Return(nil)
				return mockRepo
			},
			expectErr: false,
		},
		{
			name:      "record access with baidu referer",
			shortCode: "ABCD",
			clientIP:  "192.168.1.1",
			userAgent: "Mozilla/5.0",
			referer:   "https://www.baidu.com/s",
			setupMock: func(ctrl *gomock.Controller) *mocks.MockRedisRepositoryInterface {
				mockRepo := mocks.NewMockRedisRepositoryInterface(ctrl)
				mockRepo.EXPECT().IncrementPV(gomock.Any(), "ABCD").Return(int64(1), nil)
				mockRepo.EXPECT().AddUV(gomock.Any(), "ABCD", gomock.Any()).Return(true, nil)
				mockRepo.EXPECT().AddSource(gomock.Any(), "ABCD", "baidu").Return(nil)
				return mockRepo
			},
			expectErr: false,
		},
		{
			name:      "record access with wechat referer",
			shortCode: "ABCD",
			clientIP:  "192.168.1.1",
			userAgent: "Mozilla/5.0",
			referer:   "https://mp.weixin.qq.com/s",
			setupMock: func(ctrl *gomock.Controller) *mocks.MockRedisRepositoryInterface {
				mockRepo := mocks.NewMockRedisRepositoryInterface(ctrl)
				mockRepo.EXPECT().IncrementPV(gomock.Any(), "ABCD").Return(int64(1), nil)
				mockRepo.EXPECT().AddUV(gomock.Any(), "ABCD", gomock.Any()).Return(true, nil)
				mockRepo.EXPECT().AddSource(gomock.Any(), "ABCD", "wechat").Return(nil)
				return mockRepo
			},
			expectErr: false,
		},
		{
			name:      "record access with www prefix removed",
			shortCode: "ABCD",
			clientIP:  "192.168.1.1",
			userAgent: "Mozilla/5.0",
			referer:   "https://www.example.com",
			setupMock: func(ctrl *gomock.Controller) *mocks.MockRedisRepositoryInterface {
				mockRepo := mocks.NewMockRedisRepositoryInterface(ctrl)
				mockRepo.EXPECT().IncrementPV(gomock.Any(), "ABCD").Return(int64(1), nil)
				mockRepo.EXPECT().AddUV(gomock.Any(), "ABCD", gomock.Any()).Return(true, nil)
				mockRepo.EXPECT().AddSource(gomock.Any(), "ABCD", "example").Return(nil)
				return mockRepo
			},
			expectErr: false,
		},
		{
			name:      "record access with subdomain",
			shortCode: "ABCD",
			clientIP:  "192.168.1.1",
			userAgent: "Mozilla/5.0",
			referer:   "https://blog.example.com",
			setupMock: func(ctrl *gomock.Controller) *mocks.MockRedisRepositoryInterface {
				mockRepo := mocks.NewMockRedisRepositoryInterface(ctrl)
				mockRepo.EXPECT().IncrementPV(gomock.Any(), "ABCD").Return(int64(1), nil)
				mockRepo.EXPECT().AddUV(gomock.Any(), "ABCD", gomock.Any()).Return(true, nil)
				mockRepo.EXPECT().AddSource(gomock.Any(), "ABCD", "example").Return(nil)
				return mockRepo
			},
			expectErr: false,
		},
		{
			name:      "record access with PV error",
			shortCode: "ABCD",
			clientIP:  "192.168.1.1",
			userAgent: "Mozilla/5.0",
			referer:   "https://google.com",
			setupMock: func(ctrl *gomock.Controller) *mocks.MockRedisRepositoryInterface {
				mockRepo := mocks.NewMockRedisRepositoryInterface(ctrl)
				mockRepo.EXPECT().IncrementPV(gomock.Any(), "ABCD").Return(int64(0), errors.New("redis error"))
				mockRepo.EXPECT().AddUV(gomock.Any(), "ABCD", gomock.Any()).Return(true, nil)
				mockRepo.EXPECT().AddSource(gomock.Any(), "ABCD", "google").Return(nil)
				return mockRepo
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := tt.setupMock(ctrl)
			svc := NewAnalyticsService(mockRepo)

			err := svc.RecordAccess(context.Background(), tt.shortCode, tt.clientIP, tt.userAgent, tt.referer)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAnalyticsService_GetStats(t *testing.T) {
	tests := []struct {
		name      string
		shortCode string
		setupMock func(*gomock.Controller) *mocks.MockRedisRepositoryInterface
		expected  *model.Stats
	}{
		{
			name:      "get stats successfully",
			shortCode: "ABCD",
			setupMock: func(ctrl *gomock.Controller) *mocks.MockRedisRepositoryInterface {
				mockRepo := mocks.NewMockRedisRepositoryInterface(ctrl)
				mockRepo.EXPECT().GetPV(gomock.Any(), "ABCD").Return(int64(1000), nil)
				mockRepo.EXPECT().GetUV(gomock.Any(), "ABCD").Return(int64(500), nil)
				return mockRepo
			},
			expected: &model.Stats{PV: 1000, UV: 500},
		},
		{
			name:      "get stats with PV error",
			shortCode: "ABCD",
			setupMock: func(ctrl *gomock.Controller) *mocks.MockRedisRepositoryInterface {
				mockRepo := mocks.NewMockRedisRepositoryInterface(ctrl)
				mockRepo.EXPECT().GetPV(gomock.Any(), "ABCD").Return(int64(0), errors.New("redis error"))
				mockRepo.EXPECT().GetUV(gomock.Any(), "ABCD").Return(int64(0), nil)
				return mockRepo
			},
			expected: &model.Stats{PV: 0, UV: 0},
		},
		{
			name:      "get stats with UV error",
			shortCode: "ABCD",
			setupMock: func(ctrl *gomock.Controller) *mocks.MockRedisRepositoryInterface {
				mockRepo := mocks.NewMockRedisRepositoryInterface(ctrl)
				mockRepo.EXPECT().GetPV(gomock.Any(), "ABCD").Return(int64(1000), nil)
				mockRepo.EXPECT().GetUV(gomock.Any(), "ABCD").Return(int64(0), errors.New("redis error"))
				return mockRepo
			},
			expected: &model.Stats{PV: 1000, UV: 0},
		},
		{
			name:      "get stats with zero values",
			shortCode: "ABCD",
			setupMock: func(ctrl *gomock.Controller) *mocks.MockRedisRepositoryInterface {
				mockRepo := mocks.NewMockRedisRepositoryInterface(ctrl)
				mockRepo.EXPECT().GetPV(gomock.Any(), "ABCD").Return(int64(0), nil)
				mockRepo.EXPECT().GetUV(gomock.Any(), "ABCD").Return(int64(0), nil)
				return mockRepo
			},
			expected: &model.Stats{PV: 0, UV: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := tt.setupMock(ctrl)
			svc := NewAnalyticsService(mockRepo)

			result, err := svc.GetStats(context.Background(), tt.shortCode)

			assert.NoError(t, err)
			assert.Equal(t, tt.expected.PV, result.PV)
			assert.Equal(t, tt.expected.UV, result.UV)
		})
	}
}

func TestAnalyticsService_GetAnalytics(t *testing.T) {
	tests := []struct {
		name        string
		shortCode   string
		setupMock   func(*gomock.Controller) *mocks.MockRedisRepositoryInterface
		wantPV      int64
		wantUV      int64
		wantSourcesLen int
	}{
		{
			name:      "get analytics with sources",
			shortCode: "ABCD",
			setupMock: func(ctrl *gomock.Controller) *mocks.MockRedisRepositoryInterface {
				mockRepo := mocks.NewMockRedisRepositoryInterface(ctrl)
				mockRepo.EXPECT().GetPV(gomock.Any(), "ABCD").Return(int64(1000), nil)
				mockRepo.EXPECT().GetUV(gomock.Any(), "ABCD").Return(int64(500), nil)
				mockRepo.EXPECT().GetSources(gomock.Any(), "ABCD").Return(map[string]int64{
					"google": 500,
					"direct": 300,
					"baidu":  200,
				}, nil)
				return mockRepo
			},
			wantPV:        1000,
			wantUV:        500,
			wantSourcesLen: 3,
		},
		{
			name:      "get analytics with empty sources",
			shortCode: "ABCD",
			setupMock: func(ctrl *gomock.Controller) *mocks.MockRedisRepositoryInterface {
				mockRepo := mocks.NewMockRedisRepositoryInterface(ctrl)
				mockRepo.EXPECT().GetPV(gomock.Any(), "ABCD").Return(int64(100), nil)
				mockRepo.EXPECT().GetUV(gomock.Any(), "ABCD").Return(int64(50), nil)
				mockRepo.EXPECT().GetSources(gomock.Any(), "ABCD").Return(map[string]int64{}, nil)
				return mockRepo
			},
			wantPV:        100,
			wantUV:        50,
			wantSourcesLen: 0,
		},
		{
			name:      "get analytics with sources error",
			shortCode: "ABCD",
			setupMock: func(ctrl *gomock.Controller) *mocks.MockRedisRepositoryInterface {
				mockRepo := mocks.NewMockRedisRepositoryInterface(ctrl)
				mockRepo.EXPECT().GetPV(gomock.Any(), "ABCD").Return(int64(100), nil)
				mockRepo.EXPECT().GetUV(gomock.Any(), "ABCD").Return(int64(50), nil)
				mockRepo.EXPECT().GetSources(gomock.Any(), "ABCD").Return(nil, errors.New("redis error"))
				return mockRepo
			},
			wantPV:        100,
			wantUV:        50,
			wantSourcesLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := tt.setupMock(ctrl)
			svc := NewAnalyticsService(mockRepo)

			result, err := svc.GetAnalytics(context.Background(), tt.shortCode)

			assert.NoError(t, err)
			assert.Equal(t, tt.shortCode, result.ShortCode)
			assert.Equal(t, tt.wantPV, result.PV)
			assert.Equal(t, tt.wantUV, result.UV)
			assert.Len(t, result.TopSources, tt.wantSourcesLen)
		})
	}
}

func TestAnalyticsService_extractSource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRedisRepositoryInterface(ctrl)
	svc := NewAnalyticsService(mockRepo)

	tests := []struct {
		name     string
		referer  string
		expected string
	}{
		{
			name:     "empty referer",
			referer:  "",
			expected: "direct",
		},
		{
			name:     "google",
			referer:  "https://www.google.com/search",
			expected: "google",
		},
		{
			name:     "baidu",
			referer:  "https://www.baidu.com/s",
			expected: "baidu",
		},
		{
			name:     "bing",
			referer:  "https://www.bing.com/search",
			expected: "bing",
		},
		{
			name:     "weibo",
			referer:  "https://weibo.com/123",
			expected: "weibo",
		},
		{
			name:     "wechat",
			referer:  "https://mp.weixin.qq.com/s",
			expected: "wechat",
		},
		{
			name:     "qq",
			referer:  "https://www.qq.com",
			expected: "qq",
		},
		{
			name:     "zhihu",
			referer:  "https://www.zhihu.com",
			expected: "zhihu",
		},
		{
			name:     "invalid URL",
			referer:  "://invalid",
			expected: "unknown",
		},
		{
			name:     "custom domain",
			referer:  "https://custom.com",
			expected: "custom",
		},
		{
			name:     "subdomain",
			referer:  "https://blog.example.com",
			expected: "example",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.extractSource(tt.referer)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAnalyticsService_getTopSources(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRedisRepositoryInterface(ctrl)
	svc := NewAnalyticsService(mockRepo)

	tests := []struct {
		name     string
		sources  map[string]int64
		limit    int
		expected []model.SourceStat
	}{
		{
			name: "sort and limit sources",
			sources: map[string]int64{
				"google": 500,
				"direct": 300,
				"baidu":  200,
				"bing":   100,
			},
			limit: 2,
			expected: []model.SourceStat{
				{Source: "google", Count: 500},
				{Source: "direct", Count: 300},
			},
		},
		{
			name:    "empty sources",
			sources: map[string]int64{},
			limit:   10,
			expected: []model.SourceStat{},
		},
		{
			name: "limit greater than sources",
			sources: map[string]int64{
				"google": 500,
				"direct": 300,
			},
			limit: 10,
			expected: []model.SourceStat{
				{Source: "google", Count: 500},
				{Source: "direct", Count: 300},
			},
		},
		{
			name: "no limit (0) - returns empty because limit 0 means limit to 0 items",
			sources: map[string]int64{
				"google": 500,
				"direct": 300,
				"baidu":  200,
			},
			limit: 0,
			expected: []model.SourceStat{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.getTopSources(tt.sources, tt.limit)
			assert.Equal(t, tt.expected, result)
		})
	}
}
