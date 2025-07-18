package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/transaction-tracker/price_service/internal/models"
)

// HealthCheck handles GET /health
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, models.SuccessResponse{
		Success: true,
		Data: gin.H{
			"status":  "healthy",
			"service": "price-service",
			"version": "1.0.0",
		},
	})
}
