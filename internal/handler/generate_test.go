package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"octopus/internal/mocks"
	"octopus/internal/model"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newTestRouter(h *GenerateHandler) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	router.POST("/api/v1/shortlink/generate", h.Generate)
	return router
}

func TestNewGenerateHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockShortLinkServiceInterface(ctrl)
	handler := NewGenerateHandler(mockService)

	assert.NotNil(t, handler)
}

func TestGenerateHandler_Generate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockShortLinkServiceInterface(ctrl)
	handler := NewGenerateHandler(mockService)
	router := newTestRouter(handler)

	t.Run("invalid JSON body", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/shortlink/generate", bytes.NewBuffer([]byte("{invalid json")))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Contains(t, resp.Message, "Invalid request")
	})

	t.Run("missing URL field", func(t *testing.T) {
		reqBody := map[string]string{
			"other": "value",
		}
		jsonBody, _ := json.Marshal(reqBody)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/shortlink/generate", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Contains(t, resp.Message, "Invalid request")
	})

	t.Run("invalid JSON type for URL", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"url": 123, // should be string
		}
		jsonBody, _ := json.Marshal(reqBody)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/shortlink/generate", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("empty URL", func(t *testing.T) {
		reqBody := map[string]string{
			"url": "",
		}
		jsonBody, _ := json.Marshal(reqBody)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/shortlink/generate", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		// Empty URL is caught by validation (400)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Contains(t, resp.Message, "Invalid request")
	})

	t.Run("valid URL success", func(t *testing.T) {
		reqBody := map[string]string{
			"url": "https://example.com",
		}
		jsonBody, _ := json.Marshal(reqBody)

		mockService.EXPECT().Generate(gomock.Any(), gomock.Any()).Return(&model.GenerateResponse{
			ShortLink:   "https://s.example.com/ABCD",
			ShortCode:   "ABCD",
			OriginalURL: "https://example.com",
		}, nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/shortlink/generate", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp Response
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, 0, resp.Code)
		assert.Equal(t, "success", resp.Message)
	})

	t.Run("valid URL with expire_at", func(t *testing.T) {
		reqBody := map[string]string{
			"url":       "https://example.com",
			"expire_at": "2025-12-31T23:59:59Z",
		}
		jsonBody, _ := json.Marshal(reqBody)

		mockService.EXPECT().Generate(gomock.Any(), gomock.Any()).Return(&model.GenerateResponse{
			ShortLink:   "https://s.example.com/ABCD",
			ShortCode:   "ABCD",
			OriginalURL: "https://example.com",
		}, nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/shortlink/generate", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("service returns error", func(t *testing.T) {
		reqBody := map[string]string{
			"url": "https://example.com",
		}
		jsonBody, _ := json.Marshal(reqBody)

		mockService.EXPECT().Generate(gomock.Any(), gomock.Any()).Return(nil, assert.AnError)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/shortlink/generate", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("with params", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"url":    "https://example.com",
			"params": map[string]string{"utm_source": "google"},
		}
		jsonBody, _ := json.Marshal(reqBody)

		mockService.EXPECT().Generate(gomock.Any(), gomock.Any()).Return(&model.GenerateResponse{
			ShortLink:   "https://s.example.com/ABCD",
			ShortCode:   "ABCD",
			OriginalURL: "https://example.com",
		}, nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/shortlink/generate", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestResponse(t *testing.T) {
	t.Run("success response", func(t *testing.T) {
		resp := Response{
			Code:    0,
			Message: "success",
			Data:    "test data",
		}

		jsonBytes, err := json.Marshal(resp)
		require.NoError(t, err)

		var unmarshaled Response
		err = json.Unmarshal(jsonBytes, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, 0, unmarshaled.Code)
		assert.Equal(t, "success", unmarshaled.Message)
		assert.Equal(t, "test data", unmarshaled.Data)
	})

	t.Run("response without data", func(t *testing.T) {
		resp := Response{
			Code:    0,
			Message: "success",
		}

		jsonBytes, err := json.Marshal(resp)
		require.NoError(t, err)

		var unmarshaled Response
		err = json.Unmarshal(jsonBytes, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, 0, unmarshaled.Code)
		assert.Equal(t, "success", unmarshaled.Message)
		assert.Empty(t, unmarshaled.Data)
	})

	t.Run("response with GenerateResponse data", func(t *testing.T) {
		data := &model.GenerateResponse{
			ShortLink:   "https://s.example.com/ABCD",
			ShortCode:   "ABCD",
			OriginalURL: "https://example.com",
		}

		resp := Response{
			Code:    0,
			Message: "success",
			Data:    data,
		}

		jsonBytes, err := json.Marshal(resp)
		require.NoError(t, err)

		var unmarshaled Response
		err = json.Unmarshal(jsonBytes, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, 0, unmarshaled.Code)
		assert.Equal(t, "success", unmarshaled.Message)
		assert.NotNil(t, unmarshaled.Data)
	})

	t.Run("error response", func(t *testing.T) {
		resp := ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid request",
		}

		jsonBytes, err := json.Marshal(resp)
		require.NoError(t, err)

		var unmarshaled ErrorResponse
		err = json.Unmarshal(jsonBytes, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, unmarshaled.Code)
		assert.Equal(t, "Invalid request", unmarshaled.Message)
	})

	t.Run("error response with custom code", func(t *testing.T) {
		resp := ErrorResponse{
			Code:    1001,
			Message: "Custom error message",
		}

		jsonBytes, err := json.Marshal(resp)
		require.NoError(t, err)

		var unmarshaled ErrorResponse
		err = json.Unmarshal(jsonBytes, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, 1001, unmarshaled.Code)
		assert.Equal(t, "Custom error message", unmarshaled.Message)
	})
}

func TestGenerateRequest(t *testing.T) {
	t.Run("valid request", func(t *testing.T) {
		req := model.GenerateRequest{
			URL: "https://example.com",
		}

		jsonBytes, err := json.Marshal(req)
		require.NoError(t, err)

		var unmarshaled model.GenerateRequest
		err = json.Unmarshal(jsonBytes, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, "https://example.com", unmarshaled.URL)
	})

	t.Run("request with params", func(t *testing.T) {
		params := map[string]interface{}{"utm_source": "google", "utm_campaign": "test"}
		req := model.GenerateRequest{
			URL:    "https://example.com",
			Params: params,
		}

		jsonBytes, err := json.Marshal(req)
		require.NoError(t, err)

		var unmarshaled model.GenerateRequest
		err = json.Unmarshal(jsonBytes, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, "https://example.com", unmarshaled.URL)
		assert.Equal(t, "google", unmarshaled.Params["utm_source"])
	})
}
