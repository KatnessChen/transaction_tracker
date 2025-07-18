package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/transaction-tracker/price_service/internal/cache"
	"github.com/transaction-tracker/price_service/internal/config"
	"github.com/transaction-tracker/price_service/internal/models"
	"github.com/transaction-tracker/price_service/internal/provider"
)

type PriceHandler struct {
	cache    *cache.Service
	provider provider.StockPriceProvider
	config   *config.Config
}

func NewPriceHandler(cache *cache.Service, provider provider.StockPriceProvider, config *config.Config) *PriceHandler {
	return &PriceHandler{
		cache:    cache,
		provider: provider,
		config:   config,
	}
}

// GetCurrentPrices handles GET /api/v1/price/current/symbols
func (h *PriceHandler) GetCurrentPrices(c *gin.Context) {
	symbolsParam := c.Query("symbols")
	if symbolsParam == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Success: false,
			Error: models.ErrorDetail{
				Code:    models.ErrInvalidInput,
				Message: "symbols parameter is required",
			},
		})
		return
	}

	symbols := strings.Split(symbolsParam, ",")
	if len(symbols) > h.config.Cache.MaxSymbolsPerReq {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Success: false,
			Error: models.ErrorDetail{
				Code:    models.ErrInvalidInput,
				Message: "too many symbols requested",
			},
		})
		return
	}

	// Clean and validate symbols
	var validSymbols []string
	for _, symbol := range symbols {
		cleanSymbol := strings.TrimSpace(strings.ToUpper(symbol))
		if cleanSymbol != "" {
			validSymbols = append(validSymbols, cleanSymbol)
		}
	}

	if len(validSymbols) == 0 {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Success: false,
			Error: models.ErrorDetail{
				Code:    models.ErrInvalidInput,
				Message: "no valid symbols provided",
			},
		})
		return
	}

	var result []models.SymbolCurrentPrice
	var missingSymbols []string

	// Check cache first
	for _, symbol := range validSymbols {
		cached, err := h.cache.GetCurrentPrice(c.Request.Context(), symbol)
		if err != nil {
			// Log error but continue
			missingSymbols = append(missingSymbols, symbol)
		} else if cached != nil {
			result = append(result, *cached)
		} else {
			missingSymbols = append(missingSymbols, symbol)
		}
	}

	// Fetch missing symbols from provider
	if len(missingSymbols) > 0 {
		fetchedPrices, err := h.provider.GetCurrentPrices(c.Request.Context(), missingSymbols)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{
				Success: false,
				Error: models.ErrorDetail{
					Code:    models.ErrServiceUnavailable,
					Message: "failed to fetch price data",
				},
			})
			return
		}

		// Cache the fetched prices and add to result
		for _, price := range fetchedPrices {
			if err := h.cache.SetCurrentPrice(c.Request.Context(), price.Symbol, &price); err != nil {
				// Log error but continue
			}
			result = append(result, price)
		}
	}

	c.JSON(http.StatusOK, models.SuccessResponse{
		Success:   true,
		Data:      result,
		Timestamp: time.Now(),
	})
}

// GetHistoricalPrices handles GET /api/v1/price/historical/symbol
func (h *PriceHandler) GetHistoricalPrices(c *gin.Context) {
	symbol := strings.TrimSpace(strings.ToUpper(c.Query("symbol")))
	if symbol == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Success: false,
			Error: models.ErrorDetail{
				Code:    models.ErrInvalidInput,
				Message: "symbol parameter is required",
			},
		})
		return
	}

	resolutionStr := c.DefaultQuery("resolution", "daily")

	// Validate resolution
	resolution := models.Resolution(resolutionStr)
	if resolution != models.ResolutionDaily &&
		resolution != models.ResolutionWeekly &&
		resolution != models.ResolutionMonthly {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Success: false,
			Error: models.ErrorDetail{
				Code:    models.ErrInvalidInput,
				Message: "invalid resolution (daily, weekly, monthly allowed)",
			},
		})
		return
	}

	// Check cache first
	cached, err := h.cache.GetHistoricalPrice(c.Request.Context(), symbol, resolution)
	if err == nil && cached != nil {
		c.JSON(http.StatusOK, models.SuccessResponse{
			Success: true,
			Data:    cached,
		})
		return
	}

	// Fetch from provider
	historicalData, err := h.provider.GetHistoricalPrices(c.Request.Context(), symbol, resolution)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{
			Success: false,
			Error: models.ErrorDetail{
				Code:    models.ErrServiceUnavailable,
				Message: "failed to fetch historical data",
			},
		})
		return
	}

	// Cache the result
	if err := h.cache.SetHistoricalPrice(c.Request.Context(), symbol, resolution, historicalData); err != nil {
		// Log error but continue
	}

	c.JSON(http.StatusOK, models.SuccessResponse{
		Success:   true,
		Data:      historicalData,
		Timestamp: time.Now(),
	})
}
