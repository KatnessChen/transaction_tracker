package provider

import (
	"fmt"
	"strings"

	"github.com/transaction-tracker/price_service/internal/config"
)

// ProviderType represents different stock price providers
type ProviderType string

const (
	ProviderAlphaVantage ProviderType = "alpha_vantage"
	// Future providers can be added here
	// ProviderYahoo       ProviderType = "yahoo_finance"
	// ProviderFinnhub     ProviderType = "finnhub"
)

// NewProvider creates a new stock price provider based on configuration
func NewProvider(cfg *config.Config) (StockPriceProvider, error) {
	providerType := ProviderType(strings.ToLower(cfg.StockAPI.Provider))

	switch providerType {
	case ProviderAlphaVantage, "":
		if cfg.StockAPI.APIKey == "" {
			return nil, fmt.Errorf("API key is required for Alpha Vantage provider")
		}
		return NewAlphaVantageProvider(cfg.StockAPI.APIKey), nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s. Only 'alpha_vantage' is currently supported", providerType)
	}
}
