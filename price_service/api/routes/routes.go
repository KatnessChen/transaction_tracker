package routes

import (
	"os"

	"github.com/gin-gonic/gin"
	"github.com/transaction-tracker/price_service/api/handlers"
	"github.com/transaction-tracker/price_service/api/middlewares"
	"github.com/transaction-tracker/price_service/internal/cache"
	"github.com/transaction-tracker/price_service/internal/config"
	"github.com/transaction-tracker/price_service/internal/provider"
)

func SetupRouter(cfg *config.Config) *gin.Engine {
	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	cacheService, err := cache.NewService(cfg)
	if err != nil {
		panic("Failed to initialize cache service: " + err.Error())
	}

	stockProvider, err := provider.NewProvider(cfg)
	if err != nil {
		panic("Failed to initialize stock price provider: " + err.Error())
	}

	priceHandler := handlers.NewPriceHandler(cacheService, stockProvider, cfg)
	cacheHandler := handlers.NewCacheHandler(cacheService)

	rateLimiter := middlewares.NewRateLimiter(cfg.RateLimit.RequestsPerWindow, cfg.RateLimit.WindowDuration)

	// Create router
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(middlewares.CORSMiddleware())
	router.Use(rateLimiter.RateLimitMiddleware())

	// Health check endpoint (no auth required)
	router.GET("/health", handlers.HealthCheck)

	// API routes with authentication
	api := router.Group("/api/v1")
	api.Use(middlewares.APIKeyMiddleware(cfg.Server.APIKey))

	// Price endpoints
	priceGroup := api.Group("/price")
	{
		priceGroup.GET("/current", priceHandler.GetCurrentPrices)
		priceGroup.GET("/historical", priceHandler.GetHistoricalPrices)
	}

	// Cache management endpoints
	api.POST("/invalid-cache", cacheHandler.InvalidateCache)

	return router
}
