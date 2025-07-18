package middlewares

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/transaction-tracker/price_service/internal/models"
)

// APIKeyMiddleware validates the API key from X-API-Key header
func APIKeyMiddleware(validAPIKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip authentication for requests from localhost
		clientIP := c.ClientIP()
		if clientIP == "127.0.0.1" || clientIP == "::1" {
			c.Next()
			return
		}

		// Always block if no API key is configured
		if validAPIKey == "" {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Success: false,
				Error: models.ErrorDetail{
					Code:    models.ErrUnauthorized,
					Message: "API key is not configured on server",
				},
			})
			c.Abort()
			return
		}

		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Success: false,
				Error: models.ErrorDetail{
					Code:    models.ErrUnauthorized,
					Message: "API key is required",
				},
			})
			c.Abort()
			return
		}

		if strings.TrimSpace(apiKey) != validAPIKey {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Success: false,
				Error: models.ErrorDetail{
					Code:    models.ErrUnauthorized,
					Message: "Invalid API key",
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// CORSMiddleware handles CORS headers
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-API-Key")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}

		c.Next()
	}
}
