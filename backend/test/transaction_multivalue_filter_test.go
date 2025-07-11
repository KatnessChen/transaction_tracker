package test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/transaction-tracker/backend/api/handlers"
	"github.com/transaction-tracker/backend/internal/models"
	"github.com/transaction-tracker/backend/internal/repositories"
	"github.com/transaction-tracker/backend/internal/services"
	"github.com/transaction-tracker/backend/internal/types"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestTransactionsHandler() (*handlers.TransactionsHandler, *gorm.DB, error) {
	// Create in-memory SQLite database for testing
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Auto-migrate the schema
	if err := db.AutoMigrate(&models.User{}, &models.Transaction{}); err != nil {
		return nil, nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	// Create repositories and services
	transactionRepo := repositories.NewTransactionRepository(db)
	transactionService := services.NewTransactionService(transactionRepo)

	// Create transactions handler without AI client (extraction moved to separate handler)
	transactionsHandler := handlers.NewTransactionsHandler(transactionService)

	return transactionsHandler, db, nil
}

func createTestUser(db *gorm.DB, email string) (*models.User, error) {
	user := &models.User{
		Username: "testuser",
		Email:    email,
	}
	// Set password using the model's method
	if err := user.SetPassword("test123"); err != nil {
		return nil, err
	}
	if err := db.Create(user).Error; err != nil {
		return nil, err
	}
	return user, nil
}

func createTestTransactions(db *gorm.DB, userID uint) error {
	testTransactions := []models.Transaction{
		// Different symbols
		{
			UserID:          userID,
			Symbol:          "AAPL",
			TradeType:       types.TradeTypeBuy,
			Quantity:        100,
			Price:           150.00,
			Amount:          15000.00,
			Currency:        "USD",
			Exchange:        "NASDAQ",
			Broker:          "TDAmeritrade",
			TransactionDate: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			UserID:          userID,
			Symbol:          "GOOGL",
			TradeType:       types.TradeTypeBuy,
			Quantity:        50,
			Price:           2800.00,
			Amount:          140000.00,
			Currency:        "USD",
			Exchange:        "NASDAQ",
			Broker:          "Fidelity",
			TransactionDate: time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC),
		},
		{
			UserID:          userID,
			Symbol:          "MSFT",
			TradeType:       types.TradeTypeSell,
			Quantity:        75,
			Price:           400.00,
			Amount:          30000.00,
			Currency:        "USD",
			Exchange:        "NYSE",
			Broker:          "CharlesSchwab",
			TransactionDate: time.Date(2024, 1, 17, 0, 0, 0, 0, time.UTC),
		},
		{
			UserID:          userID,
			Symbol:          "TSLA",
			TradeType:       types.TradeTypeDividend,
			Quantity:        0,
			Price:           0,
			Amount:          500.00,
			Currency:        "EUR",
			Exchange:        "NYSE",
			Broker:          "TDAmeritrade",
			TransactionDate: time.Date(2024, 1, 18, 0, 0, 0, 0, time.UTC),
		},
		{
			UserID:          userID,
			Symbol:          "BTC",
			TradeType:       types.TradeTypeBuy,
			Quantity:        0.5,
			Price:           60000.00,
			Amount:          30000.00,
			Currency:        "CAD",
			Exchange:        "Coinbase",
			Broker:          "CoinbasePro",
			TransactionDate: time.Date(2024, 1, 19, 0, 0, 0, 0, time.UTC),
		},
	}

	return db.Create(&testTransactions).Error
}

func TestMultiValueFiltering(t *testing.T) {
	handler, db, err := setupTestTransactionsHandler()
	require.NoError(t, err)

	// Create test user
	user, err := createTestUser(db, "test@example.com")
	require.NoError(t, err)

	// Create test transactions
	err = createTestTransactions(db, user.ID)
	require.NoError(t, err)

	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Add middleware to set user_id (simulating JWT middleware)
	router.Use(func(c *gin.Context) {
		c.Set("user_id", user.ID)
		c.Next()
	})

	router.GET("/transaction-history", handler.GetTransactionHistory)

	t.Run("Filter by multiple symbols", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/transaction-history?symbol=AAPL,GOOGL,MSFT", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response handlers.GetTransactionsResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response.Success)
		assert.Equal(t, 3, len(response.Data.Transactions))

		// Verify we got the correct symbols
		symbols := make(map[string]bool)
		for _, transaction := range response.Data.Transactions {
			symbols[transaction.Symbol] = true
		}
		assert.True(t, symbols["AAPL"])
		assert.True(t, symbols["GOOGL"])
		assert.True(t, symbols["MSFT"])
		assert.False(t, symbols["TSLA"]) // Should not be included
		assert.False(t, symbols["BTC"])  // Should not be included

		// Verify filters applied response
		assert.Equal(t, []string{"AAPL", "GOOGL", "MSFT"}, response.Data.FiltersApplied.Symbols)
	})

	t.Run("Filter by multiple trade types", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/transaction-history?type=Buy,Sell", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response handlers.GetTransactionsResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response.Success)
		assert.Equal(t, 4, len(response.Data.Transactions)) // 3 Buy + 1 Sell

		// Verify we got the correct types
		tradeTypes := make(map[types.TradeType]bool)
		for _, transaction := range response.Data.Transactions {
			tradeTypes[transaction.TradeType] = true
		}
		assert.True(t, tradeTypes[types.TradeTypeBuy])
		assert.True(t, tradeTypes[types.TradeTypeSell])
		assert.False(t, tradeTypes[types.TradeTypeDividend]) // Should not be included

		// Verify filters applied response
		assert.Equal(t, []string{"Buy", "Sell"}, response.Data.FiltersApplied.TradeTypes)
	})

	t.Run("Filter by multiple exchanges", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/transaction-history?exchange=NASDAQ,NYSE", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response handlers.GetTransactionsResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response.Success)
		assert.Equal(t, 4, len(response.Data.Transactions)) // 2 NASDAQ + 2 NYSE

		// Verify we got the correct exchanges
		exchanges := make(map[string]bool)
		for _, transaction := range response.Data.Transactions {
			exchanges[transaction.Exchange] = true
		}
		assert.True(t, exchanges["NASDAQ"])
		assert.True(t, exchanges["NYSE"])
		assert.False(t, exchanges["Coinbase"]) // Should not be included

		// Verify filters applied response
		assert.Equal(t, []string{"NASDAQ", "NYSE"}, response.Data.FiltersApplied.Exchanges)
	})

	t.Run("Filter by multiple brokers", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/transaction-history?broker=TDAmeritrade,Fidelity", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response handlers.GetTransactionsResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response.Success)
		assert.Equal(t, 3, len(response.Data.Transactions)) // 2 TDAmeritrade + 1 Fidelity

		// Verify we got the correct brokers
		brokers := make(map[string]bool)
		for _, transaction := range response.Data.Transactions {
			brokers[transaction.Broker] = true
		}
		assert.True(t, brokers["TDAmeritrade"])
		assert.True(t, brokers["Fidelity"])
		assert.False(t, brokers["CharlesSchwab"]) // Should not be included
		assert.False(t, brokers["CoinbasePro"])   // Should not be included

		// Verify filters applied response
		assert.Equal(t, []string{"TDAmeritrade", "Fidelity"}, response.Data.FiltersApplied.Brokers)
	})

	t.Run("Filter by multiple currencies", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/transaction-history?currency=USD,EUR", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response handlers.GetTransactionsResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response.Success)
		assert.Equal(t, 4, len(response.Data.Transactions)) // 3 USD + 1 EUR

		// Verify we got the correct currencies
		currencies := make(map[string]bool)
		for _, transaction := range response.Data.Transactions {
			currencies[transaction.Currency] = true
		}
		assert.True(t, currencies["USD"])
		assert.True(t, currencies["EUR"])
		assert.False(t, currencies["CAD"]) // Should not be included

		// Verify filters applied response
		assert.Equal(t, []string{"USD", "EUR"}, response.Data.FiltersApplied.Currencies)
	})

	t.Run("Combined multi-value filters", func(t *testing.T) {
		// Filter for AAPL or GOOGL symbols, Buy or Sell types, and USD currency
		req := httptest.NewRequest("GET", "/transaction-history?symbol=AAPL,GOOGL&type=Buy,Sell&currency=USD", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response handlers.GetTransactionsResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response.Success)
		assert.Equal(t, 2, len(response.Data.Transactions)) // AAPL Buy + GOOGL Buy (both USD)

		// Verify all conditions are met
		for _, transaction := range response.Data.Transactions {
			assert.Contains(t, []string{"AAPL", "GOOGL"}, transaction.Symbol)
			assert.Contains(t, []types.TradeType{types.TradeTypeBuy, types.TradeTypeSell}, transaction.TradeType)
			assert.Equal(t, "USD", transaction.Currency)
		}

		// Verify filters applied response
		assert.Equal(t, []string{"AAPL", "GOOGL"}, response.Data.FiltersApplied.Symbols)
		assert.Equal(t, []string{"Buy", "Sell"}, response.Data.FiltersApplied.TradeTypes)
		assert.Equal(t, []string{"USD"}, response.Data.FiltersApplied.Currencies)
	})

	t.Run("Empty result with restrictive filters", func(t *testing.T) {
		// Filter for symbols that don't exist but are valid format (max 10 chars)
		req := httptest.NewRequest("GET", "/transaction-history?symbol=NOEXIST,FAKE123", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response handlers.GetTransactionsResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response.Success)
		if response.Data != nil {
			assert.Equal(t, 0, len(response.Data.Transactions))
			assert.Equal(t, 0, response.Data.Pagination.TotalRecords)

			// Verify filters applied response
			assert.Equal(t, []string{"NOEXIST", "FAKE123"}, response.Data.FiltersApplied.Symbols)
		}
	})

	t.Run("Invalid multi-value filter format", func(t *testing.T) {
		// Invalid symbol format
		req := httptest.NewRequest("GET", "/transaction-history?symbol=aapl,invalid@symbol", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response handlers.GetTransactionsResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.False(t, response.Success)
		if response.Errors != nil {
			assert.Contains(t, response.Errors, "symbol")
		}
	})

	t.Run("Pagination with multi-value filters", func(t *testing.T) {
		// Get first page with page_size=2
		req := httptest.NewRequest("GET", "/transaction-history?symbol=AAPL,GOOGL,MSFT&page=1&page_size=2", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response handlers.GetTransactionsResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response.Success)
		assert.Equal(t, 2, len(response.Data.Transactions))
		assert.Equal(t, 3, response.Data.Pagination.TotalRecords)
		assert.Equal(t, 2, response.Data.Pagination.TotalPages)
		assert.True(t, response.Data.Pagination.HasNext)
		assert.False(t, response.Data.Pagination.HasPrevious)
	})
}
