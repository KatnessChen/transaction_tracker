package constants

import "github.com/transaction-tracker/backend/internal/types"

// Default Configuration
const (
	DefaultAIModel    = "gemini-2.0-flash"
	DefaultAPIKey     = "GEMINI_API_KEY"
	DefaultAITimeout  = 30
	DefaultAIMaxRetry = 3
	DefaultServerAddr = ":8080"
	DefaultJWTExpiry  = 24
)

// API Routes and Endpoints
const (
	APIVersion                 = "/api/v1"
	HealthEndpoint             = "/health"
	DatabaseHealthEndpoint     = "/health/database"
	LoginEndpoint              = "/login"
	SignupEndpoint             = "/signup"
	LogoutEndpoint             = "/logout"
	MeEndpoint                 = "/me"
	HelloWorldEndpoint         = "/hello-world"
	ExtractTransEndpoint       = "/extract-transactions"
	TransactionHistoryEndpoint = "/transaction-history"
)

// Portfolio Endpoints
const (
	PortfolioSummaryEndpoint               = "/portfolio/summary"
	PortfolioHoldingsEndpoint              = "/portfolio/holdings"
	PortfolioSingleHoldingEndpoint         = "/portfolio/holdings/:symbol"
	PortfolioHistoricalMarketValueEndpoint = "/portfolio/chart/historical-market-value"
)

// HTTP Headers
const (
	AuthorizationHeader = "Authorization"
	ContentTypeHeader   = "Content-Type"
	BearerTokenPrefix   = "Bearer"
)

// MIME Types
const (
	MimeTypePNG  = "image/png"
	MimeTypeJPEG = "image/jpeg"
	MimeTypeGIF  = "image/gif"
	MimeTypeWebP = "image/webp"

	MimeTypeJSON = "application/json"
	MimeTypeForm = "multipart/form-data"
)

// Error Messages
const (
	ErrMsgAuthHeaderRequired   = "Authorization header required"
	ErrMsgInvalidAuthFormat    = "Invalid Authorization header format"
	ErrMsgInvalidToken         = "Invalid token"
	ErrMsgTokenExpired         = "Token expired"
	ErrMsgInvalidSigningMethod = "Invalid signing method"

	ErrMsgNoImagesProvided      = "No images provided"
	ErrMsgImageProcessingFailed = "Image processing failed"
	ErrMsgAIRequestFailed       = "AI request failed"

	ErrMsgInvalidTradeType  = "Invalid trade type, must be Buy, Sell, or Dividends"
	ErrMsgTickerRequired    = "Ticker should not be empty"
	ErrMsgTradeDateRequired = "TradeDate should not be empty"
	ErrMsgTradeTypeRequired = "TradeType should not be empty"
	ErrMsgNegativePrice     = "Price should not be negative"

	ErrMsgInternalServer  = "Internal server error"
	ErrMsgBadRequest      = "Bad request"
	ErrMsgUnauthorized    = "Unauthorized"
	ErrMsgForbidden       = "Forbidden"
	ErrMsgNotFound        = "Not found"
	ErrMsgTooManyRequests = "Too many requests"
)

// Success Messages
const (
	MsgTransactionsExtracted = "Transactions extracted successfully"
	MsgHealthCheckOK         = "API is healthy"
	MsgLoginSuccessful       = "Login successful"
	MsgHelloWorld            = "Hello, World! You are authenticated."
)

// Rate Limiting
const (
	DefaultRateLimit       = 100
	DefaultRateLimitWindow = 60
)

// File Upload Limits
const (
	MaxFileSize      = 10 << 20
	MaxFilesPerBatch = 10
)

// ValidTradeTypes returns a slice of valid trade types
func ValidTradeTypes() []string {
	return []string{
		string(types.TradeTypeBuy),
		string(types.TradeTypeSell),
		string(types.TradeTypeDividend),
	}
}

// ValidTradeTypesMap returns a map of valid trade types for quick lookup
func ValidTradeTypesMap() map[string]bool {
	return map[string]bool{
		string(types.TradeTypeBuy):      true,
		string(types.TradeTypeSell):     true,
		string(types.TradeTypeDividend): true,
	}
}

// SupportedImageMimeTypes returns a slice of supported image MIME types
func SupportedImageMimeTypes() []string {
	return []string{MimeTypePNG, MimeTypeJPEG, MimeTypeGIF, MimeTypeWebP}
}

// SupportedImageMimeTypesMap returns a map of supported image MIME types for quick lookup
func SupportedImageMimeTypesMap() map[string]bool {
	return map[string]bool{
		MimeTypePNG:  true,
		MimeTypeJPEG: true,
		MimeTypeGIF:  true,
		MimeTypeWebP: true,
	}
}
