package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestRecovery(t *testing.T) {
	t.Run("recovers from panic", func(t *testing.T) {
		router := gin.New()
		router.Use(Recovery())
		router.GET("/test", func(c *gin.Context) {
			panic("test panic")
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("handles normal request without panic", func(t *testing.T) {
		router := gin.New()
		router.Use(Recovery())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("recovers from nil pointer panic", func(t *testing.T) {
		router := gin.New()
		router.Use(Recovery())
		router.GET("/test", func(c *gin.Context) {
			var ptr *string
			_ = *ptr // nil pointer dereference
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("response contains error message", func(t *testing.T) {
		router := gin.New()
		router.Use(Recovery())
		router.GET("/test", func(c *gin.Context) {
			panic("test panic message")
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)

		assert.Contains(t, w.Body.String(), "Internal server error")
		assert.Contains(t, w.Body.String(), "500")
	})
}
