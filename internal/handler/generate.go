package handler

import (
	"net/http"

	"octopus/internal/model"
	"octopus/internal/service"

	"github.com/gin-gonic/gin"
)

// GenerateHandler handles short link generation
type GenerateHandler struct {
	service service.ShortLinkServiceInterface
}

// NewGenerateHandler creates a new GenerateHandler
func NewGenerateHandler(service service.ShortLinkServiceInterface) *GenerateHandler {
	return &GenerateHandler{service: service}
}

// Generate handles POST /api/v1/shortlink/generate
// @Summary Generate a short link
// @Description Generates a short link for the given URL
// @Tags shortlink
// @Accept json
// @Produce json
// @Param request body model.GenerateRequest true "Generate request"
// @Success 200 {object} Response{data=model.GenerateResponse}
// @Router /api/v1/shortlink/generate [post]
func (h *GenerateHandler) Generate(c *gin.Context) {
	var req model.GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	resp, err := h.service.Generate(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Failed to generate short link: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    resp,
	})
}

// Response is the standard API response
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ErrorResponse is the error API response
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
