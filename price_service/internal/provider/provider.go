package provider

import (
	"context"

	"github.com/transaction-tracker/price_service/internal/models"
)

// StockPriceProvider defines the interface for stock price data providers
type StockPriceProvider interface {
	// GetCurrentPrices retrieves current prices for multiple symbols
	GetCurrentPrices(ctx context.Context, symbols []string) ([]models.SymbolCurrentPrice, error)

	// GetHistoricalPrices retrieves historical prices for a single symbol
	GetHistoricalPrices(ctx context.Context, symbol string, resolution models.Resolution) (*models.SymbolHistoricalPrice, error)

	// ValidateSymbol checks if a symbol is valid
	ValidateSymbol(ctx context.Context, symbol string) bool
}
