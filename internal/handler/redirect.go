package handler

import (
	"net/http"
	"time"

	"octopus/internal/mq"
	"octopus/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// RedirectHandler handles short link redirection
type RedirectHandler struct {
	shortLinkService service.ShortLinkServiceInterface
	analyticsService service.AnalyticsServiceInterface
	mqProducer       mq.ProducerInterface
}

// NewRedirectHandler creates a new RedirectHandler
func NewRedirectHandler(
	shortLinkService service.ShortLinkServiceInterface,
	analyticsService service.AnalyticsServiceInterface,
	mqProducer mq.ProducerInterface,
) *RedirectHandler {
	return &RedirectHandler{
		shortLinkService: shortLinkService,
		analyticsService: analyticsService,
		mqProducer:       mqProducer,
	}
}

// Redirect handles GET /:shortCode
// @Summary Redirect to original URL
// @Description Redirects to the original URL for the given short code
// @Tags shortlink
// @Param shortCode path string true "Short code"
// @Success 302
// @Router /:shortCode [get]
func (h *RedirectHandler) Redirect(c *gin.Context) {
	shortCode := c.Param("shortCode")

	// Get short link
	sl, err := h.shortLinkService.Get(c.Request.Context(), shortCode)
	if err != nil {
		c.HTML(http.StatusNotFound, "404.html", gin.H{
			"code": shortCode,
		})
		return
	}

	// Expand URL with query params
	queryParams := make(map[string]string)
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			queryParams[key] = values[0]
		}
	}

	targetURL, err := h.shortLinkService.ExpandURL(c.Request.Context(), shortCode, queryParams)
	if err != nil {
		targetURL = sl.OriginalURL
	}

	// Record analytics
	clientIP := c.ClientIP()
	userAgent := c.Request.UserAgent()
	referer := c.Request.Header.Get("Referer")

	// Record in Redis for real-time stats
	go func() {
		if err := h.analyticsService.RecordAccess(c.Request.Context(), shortCode, clientIP, userAgent, referer); err != nil {
			log.Error().Err(err).Str("short_code", shortCode).Msg("Failed to record access")
		}
	}()

	// Send to MQ for async processing
	if h.mqProducer != nil {
		go func() {
			msg := &mq.AccessLogMessage{
				ShortCode:  shortCode,
				ClientIP:   clientIP,
				UserAgent:  userAgent,
				Referer:    referer,
				AccessTime: time.Now(),
			}
			if err := h.mqProducer.SendAccessLog(c.Request.Context(), msg); err != nil {
				log.Error().Err(err).Str("short_code", shortCode).Msg("Failed to send access log to MQ")
			}
		}()
	}

	// 302 Redirect
	c.Redirect(http.StatusFound, targetURL)
}

// GetStats handles GET /api/v1/analytics/:shortCode
// @Summary Get analytics for a short link
// @Description Returns PV/UV statistics for a short link
// @Tags analytics
// @Param shortCode path string true "Short code"
// @Success 200 {object} Response{data=service.AnalyticsResponse}
// @Router /api/v1/analytics/:shortCode [get]
func (h *RedirectHandler) GetStats(c *gin.Context) {
	shortCode := c.Param("shortCode")

	// Check if short link exists
	_, err := h.shortLinkService.Get(c.Request.Context(), shortCode)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Code:    http.StatusNotFound,
			Message: "Short link not found",
		})
		return
	}

	// Get analytics
	analytics, err := h.analyticsService.GetAnalytics(c.Request.Context(), shortCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Failed to get analytics",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    analytics,
	})
}
