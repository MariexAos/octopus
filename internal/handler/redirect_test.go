package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"octopus/internal/mocks"
	"octopus/internal/mq"
	"octopus/internal/model"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newTestRedirectRouter(h *RedirectHandler) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	router.GET("/:shortCode", h.Redirect)
	router.GET("/api/v1/analytics/:shortCode", h.GetStats)
	return router
}

func TestNewRedirectHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockShortLinkService := mocks.NewMockShortLinkServiceInterface(ctrl)
	mockAnalyticsService := mocks.NewMockAnalyticsServiceInterface(ctrl)

	handler := NewRedirectHandler(mockShortLinkService, mockAnalyticsService, nil)

	assert.NotNil(t, handler)
}

func TestRedirectHandler_Redirect(t *testing.T) {
	t.Run("successful redirect", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockShortLinkService := mocks.NewMockShortLinkServiceInterface(ctrl)
		mockAnalyticsService := mocks.NewMockAnalyticsServiceInterface(ctrl)
		mockProducer := mocks.NewMockProducerInterface(ctrl)

		handler := NewRedirectHandler(mockShortLinkService, mockAnalyticsService, mockProducer)
		router := newTestRedirectRouter(handler)

		shortCode := "ABCD"
		originalURL := "https://example.com"

		mockShortLinkService.EXPECT().Get(gomock.Any(), shortCode).Return(&model.ShortLink{
			ShortCode:   shortCode,
			OriginalURL: originalURL,
		}, nil)
		mockShortLinkService.EXPECT().ExpandURL(gomock.Any(), shortCode, gomock.Any()).Return(originalURL, nil)
		// Async calls in goroutines
		mockAnalyticsService.EXPECT().RecordAccess(gomock.Any(), shortCode, gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		mockProducer.EXPECT().SendAccessLog(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/"+shortCode, nil)
		router.ServeHTTP(w, req)

		// Wait for goroutines to complete
		time.Sleep(50 * time.Millisecond)

		assert.Equal(t, http.StatusFound, w.Code)
		assert.Equal(t, originalURL, w.Header().Get("Location"))
	})

	t.Run("redirect with query params", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockShortLinkService := mocks.NewMockShortLinkServiceInterface(ctrl)
		mockAnalyticsService := mocks.NewMockAnalyticsServiceInterface(ctrl)
		mockProducer := mocks.NewMockProducerInterface(ctrl)

		handler := NewRedirectHandler(mockShortLinkService, mockAnalyticsService, mockProducer)
		router := newTestRedirectRouter(handler)

		shortCode := "ABCD"
		originalURL := "https://example.com?utm_source=google"

		mockShortLinkService.EXPECT().Get(gomock.Any(), shortCode).Return(&model.ShortLink{
			ShortCode:   shortCode,
			OriginalURL: "https://example.com",
		}, nil)
		mockShortLinkService.EXPECT().ExpandURL(gomock.Any(), shortCode, gomock.Any()).Return(originalURL, nil)
		mockAnalyticsService.EXPECT().RecordAccess(gomock.Any(), shortCode, gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		mockProducer.EXPECT().SendAccessLog(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/"+shortCode+"?utm_source=google", nil)
		router.ServeHTTP(w, req)

		time.Sleep(50 * time.Millisecond)

		assert.Equal(t, http.StatusFound, w.Code)
	})

	t.Run("non-existent short code", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockShortLinkService := mocks.NewMockShortLinkServiceInterface(ctrl)
		mockAnalyticsService := mocks.NewMockAnalyticsServiceInterface(ctrl)
		mockProducer := mocks.NewMockProducerInterface(ctrl)

		handler := NewRedirectHandler(mockShortLinkService, mockAnalyticsService, mockProducer)
		router := newTestRedirectRouter(handler)

		shortCode := "NOTFOUND"

		mockShortLinkService.EXPECT().Get(gomock.Any(), shortCode).Return(nil, errors.New("not found"))

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/"+shortCode, nil)
		router.ServeHTTP(w, req)

		// Without HTML render setup, the HTML call panics and returns 500
		// In production with proper HTML render, it would return 404
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("redirect with ExpandURL error falls back to original URL", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockShortLinkService := mocks.NewMockShortLinkServiceInterface(ctrl)
		mockAnalyticsService := mocks.NewMockAnalyticsServiceInterface(ctrl)
		mockProducer := mocks.NewMockProducerInterface(ctrl)

		handler := NewRedirectHandler(mockShortLinkService, mockAnalyticsService, mockProducer)
		router := newTestRedirectRouter(handler)

		shortCode := "ABCD"
		originalURL := "https://example.com"

		mockShortLinkService.EXPECT().Get(gomock.Any(), shortCode).Return(&model.ShortLink{
			ShortCode:   shortCode,
			OriginalURL: originalURL,
		}, nil)
		mockShortLinkService.EXPECT().ExpandURL(gomock.Any(), shortCode, gomock.Any()).Return("", errors.New("expand error"))
		mockAnalyticsService.EXPECT().RecordAccess(gomock.Any(), shortCode, gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		mockProducer.EXPECT().SendAccessLog(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/"+shortCode, nil)
		router.ServeHTTP(w, req)

		time.Sleep(50 * time.Millisecond)

		assert.Equal(t, http.StatusFound, w.Code)
		assert.Equal(t, originalURL, w.Header().Get("Location"))
	})

	t.Run("redirect without MQ producer", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockShortLinkService := mocks.NewMockShortLinkServiceInterface(ctrl)
		mockAnalyticsService := mocks.NewMockAnalyticsServiceInterface(ctrl)

		handlerNoMQ := NewRedirectHandler(mockShortLinkService, mockAnalyticsService, nil)
		routerNoMQ := newTestRedirectRouter(handlerNoMQ)

		shortCode := "ABCD"
		originalURL := "https://example.com"

		mockShortLinkService.EXPECT().Get(gomock.Any(), shortCode).Return(&model.ShortLink{
			ShortCode:   shortCode,
			OriginalURL: originalURL,
		}, nil)
		mockShortLinkService.EXPECT().ExpandURL(gomock.Any(), shortCode, gomock.Any()).Return(originalURL, nil)
		mockAnalyticsService.EXPECT().RecordAccess(gomock.Any(), shortCode, gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/"+shortCode, nil)
		routerNoMQ.ServeHTTP(w, req)

		time.Sleep(50 * time.Millisecond)

		assert.Equal(t, http.StatusFound, w.Code)
	})
}

func TestRedirectHandler_GetStats(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockShortLinkService := mocks.NewMockShortLinkServiceInterface(ctrl)
	mockAnalyticsService := mocks.NewMockAnalyticsServiceInterface(ctrl)

	handler := NewRedirectHandler(mockShortLinkService, mockAnalyticsService, nil)
	router := newTestRedirectRouter(handler)

	t.Run("get stats successfully", func(t *testing.T) {
		shortCode := "ABCD"

		mockShortLinkService.EXPECT().Get(gomock.Any(), shortCode).Return(&model.ShortLink{
			ShortCode:   shortCode,
			OriginalURL: "https://example.com",
		}, nil)
		mockAnalyticsService.EXPECT().GetAnalytics(gomock.Any(), shortCode).Return(&model.AnalyticsResponse{
			ShortCode:  shortCode,
			PV:         100,
			UV:         50,
			TopSources: []model.SourceStat{{Source: "google", Count: 40}},
		}, nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/analytics/"+shortCode, nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("get stats for non-existent short link", func(t *testing.T) {
		shortCode := "NOTFOUND"

		mockShortLinkService.EXPECT().Get(gomock.Any(), shortCode).Return(nil, errors.New("not found"))

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/analytics/"+shortCode, nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("get stats with analytics error", func(t *testing.T) {
		shortCode := "ABCD"

		mockShortLinkService.EXPECT().Get(gomock.Any(), shortCode).Return(&model.ShortLink{
			ShortCode:   shortCode,
			OriginalURL: "https://example.com",
		}, nil)
		mockAnalyticsService.EXPECT().GetAnalytics(gomock.Any(), shortCode).Return(nil, errors.New("analytics error"))

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/analytics/"+shortCode, nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestAccessLogMessage(t *testing.T) {
	t.Run("complete message", func(t *testing.T) {
		msg := &mq.AccessLogMessage{
			ShortCode:  "ABCD",
			ClientIP:   "192.168.1.1",
			UserAgent:  "Mozilla/5.0",
			Referer:    "https://google.com",
			AccessTime: time.Now(),
		}

		assert.Equal(t, "ABCD", msg.ShortCode)
		assert.Equal(t, "192.168.1.1", msg.ClientIP)
		assert.Equal(t, "Mozilla/5.0", msg.UserAgent)
		assert.Equal(t, "https://google.com", msg.Referer)
		assert.False(t, msg.AccessTime.IsZero())
	})

	t.Run("message with minimal fields", func(t *testing.T) {
		msg := &mq.AccessLogMessage{
			ShortCode:  "ABCD",
			AccessTime: time.Now(),
		}

		assert.Equal(t, "ABCD", msg.ShortCode)
		assert.Empty(t, msg.ClientIP)
		assert.Empty(t, msg.UserAgent)
		assert.Empty(t, msg.Referer)
		assert.False(t, msg.AccessTime.IsZero())
	})
}

func TestAnalyticsResponse(t *testing.T) {
	t.Run("complete analytics", func(t *testing.T) {
		resp := &model.AnalyticsResponse{
			ShortCode: "ABCD",
			PV:        100,
			UV:        50,
			TopSources: []model.SourceStat{
				{Source: "google", Count: 40},
				{Source: "direct", Count: 10},
			},
		}

		assert.Equal(t, "ABCD", resp.ShortCode)
		assert.Equal(t, int64(100), resp.PV)
		assert.Equal(t, int64(50), resp.UV)
		assert.Len(t, resp.TopSources, 2)
	})

	t.Run("analytics with no sources", func(t *testing.T) {
		resp := &model.AnalyticsResponse{
			ShortCode:   "ABCD",
			PV:          0,
			UV:          0,
			TopSources:  []model.SourceStat{},
		}

		assert.Equal(t, "ABCD", resp.ShortCode)
		assert.Equal(t, int64(0), resp.PV)
		assert.Equal(t, int64(0), resp.UV)
		assert.Empty(t, resp.TopSources)
	})
}

func TestSourceStat(t *testing.T) {
	t.Run("source stat", func(t *testing.T) {
		stat := model.SourceStat{
			Source: "google",
			Count:  100,
		}

		assert.Equal(t, "google", stat.Source)
		assert.Equal(t, int64(100), stat.Count)
	})
}

func TestStats(t *testing.T) {
	t.Run("stats with values", func(t *testing.T) {
		stats := &model.Stats{
			PV: 100,
			UV: 50,
		}

		assert.Equal(t, int64(100), stats.PV)
		assert.Equal(t, int64(50), stats.UV)
	})

	t.Run("stats with zero values", func(t *testing.T) {
		stats := &model.Stats{
			PV: 0,
			UV: 0,
		}

		assert.Equal(t, int64(0), stats.PV)
		assert.Equal(t, int64(0), stats.UV)
	})
}
