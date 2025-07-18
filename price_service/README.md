# Price Service

A standalone microservice for providing stock price data with Redis caching and API key authentication.

## Features

- **Current Price API**: Get real-time prices for multiple symbols
- **Historical Price API**: Get historical price data with configurable resolution
- **Redis Caching**: Intelligent caching strategy with configurable TTL
- **API Key Authentication**: Secure access via X-API-Key header
- **Cache Management**: Runtime TTL updates and cache invalidation
- **Alpha Vantage Integration**: Real-time data from Alpha Vantage API

## Quick Start

### Prerequisites

- Go 1.21+
- Redis server
- Alpha Vantage API key (get free key from https://www.alphavantage.co/support/#api-key)

### Installation

1. **Clone and setup**:

   ```bash
   cd price_service
   cp .env.example .env
   ```

2. **Configure environment**:

   ```bash
   # Edit .env file
   API_KEY=your-api-key-here
   STOCK_API_PROVIDER=alpha_vantage
   STOCK_API_KEY=your-alpha-vantage-api-key
   REDIS_HOST=localhost
   REDIS_PORT=6379
   ```

3. **Install dependencies**:

   ```bash
   go mod download
   ```

4. **Run the service**:
   ```bash
   go run main.go
   ```

The service will start on port 8081 by default.

## API Endpoints

### Authentication

All API endpoints (except `/health`) require authentication via the `X-API-Key` header:

```bash
curl -H "X-API-Key: your-api-key" http://localhost:8081/api/v1/price/current/symbols?symbols=AAPL
```

### Current Prices

**GET** `/api/v1/price/current/symbols`

Get current prices for multiple symbols.

**Query Parameters:**

- `symbols` (required): Comma-separated list of stock symbols (max 50)

**Example:**

```bash
curl -H "X-API-Key: your-api-key" \
  "http://localhost:8081/api/v1/price/current/symbols?symbols=AAPL,GOOGL,MSFT"
```

**Response:**

```json
{
  "data": [
    {
      "symbol": "AAPL",
      "current_price": 175.25,
      "currency": "USD",
      "change": 2.5,
      "change_percent": 1.45,
      "previous_close": 172.75,
      "timestamp": "2025-07-17T10:30:00Z"
    }
  ],
  "timestamp": "2025-07-17T10:30:00Z"
}
```

### Historical Prices

**GET** `/api/v1/price/historical/symbol`

Get historical prices for a single symbol.

**Query Parameters:**

- `symbol` (required): Stock symbol
- `from` (required): Start date (YYYY-MM-DD)
- `to` (required): End date (YYYY-MM-DD)
- `resolution` (optional): Data resolution - `daily`, `weekly`, `monthly` (default: `daily`)

**Example:**

```bash
curl -H "X-API-Key: your-api-key" \
  "http://localhost:8081/api/v1/price/historical/symbol?symbol=AAPL&from=2025-01-01&to=2025-07-17&resolution=daily"
```

**Response:**

```json
{
  "symbol": "AAPL",
  "resolution": "daily",
  "historical_prices": [
    {
      "date": "2025-01-01",
      "price": 180.5
    },
    {
      "date": "2025-01-02",
      "price": 182.25
    }
  ]
}
```

### Cache Management

**PUT** `/api/v1/update-ttl`

Update cache TTL globally.

**Request Body:**

```json
{
  "minutes": 30
}
```

**POST** `/api/v1/invalid-cache`

Invalidate all cache entries.

**Example:**

```bash
curl -X POST -H "X-API-Key: your-api-key" \
  http://localhost:8081/api/v1/invalid-cache
```

### Health Check

**GET** `/health`

Check service health (no authentication required).

**Response:**

```json
{
  "success": true,
  "data": {
    "status": "healthy",
    "service": "price-service",
    "version": "1.0.0"
  }
}
```

### Current Prices

Get current prices for multiple symbols:

```bash
curl -H "X-API-Key: your-api-key" \
  "http://localhost:8081/api/v1/price/current/symbols?symbols=AAPL,NVDA,TSLA"
```

**Response**:

```json
{
  "success": true,
  "data": {
    "data": [
      {
        "symbol": "AAPL",
        "current_price": 150.25,
        "currency": "USD",
        "change": 2.5,
        "change_percent": 1.69,
        "previous_close": 147.75,
        "timestamp": "2025-07-17T10:30:00Z"
      }
    ],
    "timestamp": "2025-07-17T10:30:00Z"
  }
}
```

### Historical Prices

Get historical price data for a symbol:

```bash
curl -H "X-API-Key: your-api-key" \
  "http://localhost:8081/api/v1/price/historical/symbol?symbol=AAPL&from=2024-01-01&to=2024-12-31&resolution=daily"
```

**Response**:

```json
{
  "success": true,
  "data": {
    "symbol": "AAPL",
    "resolution": "daily",
    "historical_prices": [
      {
        "date": "2024-01-01",
        "price": 150.0
      }
    ]
  }
}
```

### Cache Management

**Update TTL**:

```bash
curl -X PUT -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{"minutes": 30}' \
  "http://localhost:8081/api/v1/update-ttl"
```

**Invalidate Cache**:

```bash
curl -X POST -H "X-API-Key: your-api-key" \
  "http://localhost:8081/api/v1/invalid-cache"
```

## Cache Strategy

### Current Prices

- **TTL**: 60 minutes (configurable)
- **Key Pattern**: `current-price:{symbol}`
- **Strategy**: Individual symbol caching for efficient multi-symbol requests

### Historical Prices

- **TTL**: 24 hours (fixed)
- **Key Pattern**: `historical-price:{symbol}`
- **Strategy**: Full dataset caching per symbol

## Configuration

### Environment Variables

| Variable                  | Description        | Default     |
| ------------------------- | ------------------ | ----------- |
| `PORT`                    | Server port        | `8081`      |
| `API_KEY`                 | Authentication key | `""`        |
| `REDIS_HOST`              | Redis host         | `localhost` |
| `REDIS_PORT`              | Redis port         | `6379`      |
| `DEFAULT_TTL_MINUTES`     | Cache TTL          | `60`        |
| `MAX_SYMBOLS_PER_REQUEST` | Symbol limit       | `50`        |

### Stock Provider Integration

To integrate with additional stock price providers:

1. Implement the `StockPriceProvider` interface
2. Add provider to the factory in `internal/provider/factory.go`
3. Configure provider credentials in `.env`

Example provider implementation:

```go
type AlphaVantageProvider struct {
    apiKey  string
    baseURL string
    client  *http.Client
}

func (p *AlphaVantageProvider) GetCurrentPrices(ctx context.Context, symbols []string) ([]models.SymbolCurrentPrice, error) {
    // Implementation here
}
```

## Error Handling

The service returns standardized error responses:

```json
{
  "success": false,
  "error": {
    "code": "SYMBOL_NOT_FOUND",
    "message": "Symbol INVALID not found"
  }
}
```

**Error Codes**:

- `SYMBOL_NOT_FOUND`: Invalid or unknown symbol
- `MARKET_CLOSED`: Market is currently closed
- `RATE_LIMIT_EXCEEDED`: Too many requests
- `SERVICE_UNAVAILABLE`: Upstream service error
- `INVALID_INPUT`: Invalid request parameters
- `UNAUTHORIZED`: Invalid or missing API key

## Development

### Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific test files
go test ./tests/...
```

### Adding New Providers

1. Create provider implementation in `internal/provider/`
2. Implement `StockPriceProvider` interface
3. Add configuration options
4. Update factory pattern in `factory.go`

## Production Deployment

### Docker

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o price-service main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/price-service .
CMD ["./price-service"]
```

### Health Check

```bash
curl http://localhost:8081/health
```

### Monitoring

The service provides:

- Health check endpoint
- Structured logging
- Error tracking
- Performance metrics (TODO)

## Architecture

```
price_service/
├── main.go                    # Application entry point
├── internal/
│   ├── config/               # Configuration management
│   ├── models/               # Data models and types
│   ├── cache/                # Redis cache service
│   ├── provider/             # Stock price providers
│   ├── handlers/             # HTTP handlers
│   └── middleware/           # Authentication & CORS
├── docs/                     # API documentation
└── tests/                    # Test files
```

The service follows clean architecture principles with clear separation of concerns and dependency injection.
