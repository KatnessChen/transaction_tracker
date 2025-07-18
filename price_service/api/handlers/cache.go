package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/transaction-tracker/price_service/internal/cache"
	"github.com/transaction-tracker/price_service/internal/models"
)

type CacheHandler struct {
	cache *cache.Service
}

func NewCacheHandler(cache *cache.Service) *CacheHandler {
	return &CacheHandler{cache: cache}
}

// InvalidateCache handles POST /api/v1/invalid-cache
func (h *CacheHandler) InvalidateCache(c *gin.Context) {
	if err := h.cache.InvalidateAll(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Success: false,
			Error: models.ErrorDetail{
				Code:    models.ErrServiceUnavailable,
				Message: "failed to invalidate cache",
			},
		})
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse{
		Success: true,
		Data:    gin.H{"message": "Cache invalidated successfully"},
	})
}
