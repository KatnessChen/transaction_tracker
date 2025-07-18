package models

import "time"

// Resolution types for historical data
type Resolution string

const (
	ResolutionDaily    Resolution = "daily"
	ResolutionWeekly   Resolution = "weekly"
	ResolutionMonthly  Resolution = "monthly"
	ResolutionIntraday Resolution = "intraday"
)

// SymbolCurrentPrice represents current price data for a symbol
type SymbolCurrentPrice struct {
	Symbol        string    `json:"symbol"`
	CurrentPrice  float64   `json:"current_price"`
	Currency      string    `json:"currency"`
	Change        float64   `json:"change"`         // absolute difference (current_price - previous_close)
	ChangePercent float64   `json:"change_percent"` // relative difference (change/previous_close)
	PreviousClose float64   `json:"previous_close"`
	Timestamp     time.Time `json:"timestamp"`
}

// ClosePrice represents a date-price pair
type ClosePrice struct {
	Date  string  `json:"date"` // YYYY-MM-DD format
	Price float64 `json:"price"`
}

// SymbolHistoricalPrice represents historical price data for a symbol
type SymbolHistoricalPrice struct {
	Symbol           string       `json:"symbol"`
	Resolution       Resolution   `json:"resolution"`
	HistoricalPrices []ClosePrice `json:"historical_prices"`
}

// Error response structure
type ErrorCode string

const (
	ErrSymbolNotFound     ErrorCode = "SYMBOL_NOT_FOUND"
	ErrMarketClosed       ErrorCode = "MARKET_CLOSED"
	ErrRateLimitExceeded  ErrorCode = "RATE_LIMIT_EXCEEDED"
	ErrServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
	ErrInvalidInput       ErrorCode = "INVALID_INPUT"
	ErrUnauthorized       ErrorCode = "UNAUTHORIZED"
)

// ErrorResponse represents the standard error response format
type ErrorResponse struct {
	Success bool        `json:"success"`
	Error   ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

// SuccessResponse represents successful responses
type SuccessResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

// Cache configuration update request
type UpdateTTLRequest struct {
	Minutes int `json:"minutes" binding:"required,min=1,max=1440"` // 1 minute to 24 hours
}
