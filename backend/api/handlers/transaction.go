package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/transaction-tracker/backend/internal/models"
	"github.com/transaction-tracker/backend/internal/services"
	"github.com/transaction-tracker/backend/internal/types"
	"github.com/transaction-tracker/backend/internal/utils"
)

// NewTransactionsHandler creates a new TransactionsHandler
func NewTransactionsHandler(service *services.TransactionService) *TransactionsHandler {
	return &TransactionsHandler{transactionService: service}
}

// TransactionsHandler handles transaction-related endpoints
type TransactionsHandler struct {
	transactionService *services.TransactionService
}

// TransactionRequest represents the request structure for creating transactions
type TransactionRequest struct {
	Symbol    string          `json:"symbol" binding:"required"`
	Exchange  string          `json:"exchange"`
	Broker    string          `json:"broker"`
	Currency  string          `json:"currency" binding:"required"`
	TradeDate string          `json:"transaction_date" binding:"required"`
	TradeType types.TradeType `json:"trade_type" binding:"required"`
	Quantity  float64         `json:"quantity" binding:"required,gt=0"`
	Price     float64         `json:"price" binding:"required,gt=0"`
	Amount    float64         `json:"amount" binding:"required,gt=0"`
	UserNotes string          `json:"user_notes"`
}

// CreateTransactionsRequest represents the batch request for creating transactions
type CreateTransactionsRequest struct {
	Transactions []TransactionRequest `json:"transactions" binding:"required,min=1"`
}

// CreateTransactionsResponse represents the response for creating transactions
type CreateTransactionsResponse struct {
	Success bool                    `json:"success"`
	Message string                  `json:"message"`
	Data    *CreateTransactionsData `json:"data,omitempty"`
	Errors  map[string][]string     `json:"errors,omitempty"`
}

// CreateTransactionsData represents the data part of create transactions response
type CreateTransactionsData struct {
	Transactions []types.TransactionData `json:"transactions"`
	Count        int                     `json:"count"`
}

// GetTransactionsResponse represents the response for getting transaction history
type GetTransactionsResponse struct {
	Success bool                 `json:"success"`
	Message string               `json:"message"`
	Data    *GetTransactionsData `json:"data,omitempty"`
	Errors  map[string][]string  `json:"errors,omitempty"`
}

// GetTransactionsData represents the data part of get transactions response
type GetTransactionsData struct {
	Transactions   []types.TransactionData `json:"transactions"`
	Pagination     types.PaginationData    `json:"pagination"`
	FiltersApplied FiltersApplied          `json:"filters_applied"`
}

// FiltersApplied represents the filters that were applied to the query
type FiltersApplied struct {
	Symbols    []string `json:"symbols,omitempty"`    // Support multiple symbols
	TradeTypes []string `json:"types,omitempty"`      // Support multiple types
	Exchanges  []string `json:"exchanges,omitempty"`  // Support multiple exchanges
	Brokers    []string `json:"brokers,omitempty"`    // Support multiple brokers
	Currencies []string `json:"currencies,omitempty"` // Support multiple currencies
	Timeframe  *string  `json:"timeframe,omitempty"`
}

// TransactionQueryParams represents parsed query parameters for transaction filtering
type TransactionQueryParams struct {
	Page       int
	PageSize   int
	Symbols    []string // Support multiple symbols
	TradeTypes []string // Support multiple types
	Exchanges  []string // Support multiple exchanges
	Brokers    []string // Support multiple brokers
	Currencies []string // Support multiple currencies
	StartDate  *time.Time
	EndDate    *time.Time
	SortBy     string
	SortOrder  string
}

// modelToTransactionData converts a models.Transaction to types.TransactionData
func modelToTransactionData(transaction models.Transaction) types.TransactionData {
	return types.TransactionData{
		ID:              transaction.TransactionID.String(),
		Symbol:          transaction.Symbol,
		TradeType:       types.TradeType(transaction.TradeType),
		Quantity:        transaction.Quantity,
		Price:           transaction.Price,
		Amount:          transaction.Amount,
		Currency:        transaction.Currency,
		Broker:          transaction.Broker,
		Exchange:        transaction.Exchange,
		TransactionDate: transaction.TransactionDate.Format("2006-01-02"),
		UserNotes:       transaction.UserNotes,
	}
}

// modelsToTransactionData converts a slice of models.Transaction to []types.TransactionData
func modelsToTransactionData(transactions []models.Transaction) []types.TransactionData {
	if len(transactions) == 0 {
		return []types.TransactionData{}
	}

	responseTransactions := make([]types.TransactionData, len(transactions))
	for i, transaction := range transactions {
		responseTransactions[i] = modelToTransactionData(transaction)
	}
	return responseTransactions
}

// CreateTransactions handles the creation of transaction records in batch
func (h *TransactionsHandler) CreateTransactions(c *gin.Context) {
	var req CreateTransactionsRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, CreateTransactionsResponse{
			Success: false,
			Message: "Invalid request format",
			Errors:  map[string][]string{"json": {"Invalid JSON format"}},
		})
		return
	}

	// Validate and convert request transactions to model transactions
	var validatedTransactions []models.Transaction
	var validationErrors = make(map[string][]string)

	for i, reqTransaction := range req.Transactions {
		// Validate each transaction
		if err := validateTransaction(reqTransaction); err != nil {
			validationErrors[fmt.Sprintf("transaction[%d]", i)] = []string{err.Error()}
			continue
		}

		// Parse date
		transactionDate, err := time.Parse("2006-01-02", reqTransaction.TradeDate)
		if err != nil {
			validationErrors[fmt.Sprintf("transaction[%d].transaction_date", i)] = []string{"Invalid date format, expected YYYY-MM-DD"}
			continue
		}

		// Convert request transaction to model transaction
		transaction := models.Transaction{
			TradeType:       reqTransaction.TradeType,
			Symbol:          reqTransaction.Symbol,
			Quantity:        reqTransaction.Quantity,
			Price:           reqTransaction.Price,
			Amount:          reqTransaction.Amount,
			Currency:        reqTransaction.Currency,
			Broker:          reqTransaction.Broker,
			Exchange:        reqTransaction.Exchange,
			TransactionDate: transactionDate,
			UserNotes:       reqTransaction.UserNotes,
		}

		validatedTransactions = append(validatedTransactions, transaction)
	}

	// If there were validation errors, return them
	if len(validationErrors) > 0 {
		c.JSON(http.StatusBadRequest, CreateTransactionsResponse{
			Success: false,
			Message: "Validation failed",
			Errors:  validationErrors,
		})
		return
	}

	// Get user ID from JWT context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, CreateTransactionsResponse{
			Success: false,
			Message: "Unauthorized",
			Errors:  map[string][]string{"auth": {"User not authenticated"}},
		})
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, CreateTransactionsResponse{
			Success: false,
			Message: "Invalid user ID in context",
			Errors:  map[string][]string{"auth": {"Invalid user ID in context"}},
		})
		return
	}

	// Create transactions using injected service
	createdTransactions, err := h.transactionService.CreateTransactions(userUUID, validatedTransactions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, CreateTransactionsResponse{
			Success: false,
			Message: "Failed to create transactions",
			Errors:  map[string][]string{"database": {err.Error()}},
		})
		return
	}

	// Convert created transactions to response format
	responseTransactions := modelsToTransactionData(createdTransactions)

	c.JSON(http.StatusCreated, CreateTransactionsResponse{
		Success: true,
		Message: "Transactions created successfully",
		Data: &CreateTransactionsData{
			Transactions: responseTransactions,
			Count:        len(responseTransactions),
		},
	})
}

// GetTransactionHistory handles GET /transaction-history/ endpoint
func (h *TransactionsHandler) GetTransactionHistory(c *gin.Context) {
	// Extract user ID from JWT context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, GetTransactionsResponse{
			Success: false,
			Message: "Unauthorized",
			Errors:  map[string][]string{"auth": {"User not authenticated"}},
		})
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, GetTransactionsResponse{
			Success: false,
			Message: "Invalid user ID in context",
			Errors:  map[string][]string{"auth": {"Invalid user ID in context"}},
		})
		return
	}

	// Parse and validate query parameters
	params, validationErrors := parseTransactionQueryParams(c)
	if len(validationErrors) > 0 {
		c.JSON(http.StatusBadRequest, GetTransactionsResponse{
			Success: false,
			Message: "Validation failed",
			Errors:  validationErrors,
		})
		return
	}

	// Build filter with user ID (security requirement)
	filter := services.TransactionFilter{
		UserID:         &userUUID, // Ensure user can only see their own transactions
		Symbols:        params.Symbols,
		TradeTypes:     params.TradeTypes,
		Exchanges:      params.Exchanges,
		Brokers:        params.Brokers,
		Currencies:     params.Currencies,
		StartDate:      params.StartDate,
		EndDate:        params.EndDate,
		OrderBy:        params.SortBy,
		OrderDirection: params.SortOrder,
		Limit:          params.PageSize,
		Offset:         (params.Page - 1) * params.PageSize,
	}

	// Get transactions and total count
	transactions, err := h.transactionService.GetTransactionsWithFilter(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, GetTransactionsResponse{
			Success: false,
			Message: "Failed to retrieve transactions",
			Errors:  map[string][]string{"database": {err.Error()}},
		})
		return
	}

	totalCount, err := h.transactionService.CountTransactions(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, GetTransactionsResponse{
			Success: false,
			Message: "Failed to count transactions",
			Errors:  map[string][]string{"database": {err.Error()}},
		})
		return
	}

	// Convert transactions to response format (exclude user_id for security)
	responseTransactions := modelsToTransactionData(transactions)

	// Calculate pagination
	totalPages := int((totalCount + int64(params.PageSize) - 1) / int64(params.PageSize))
	hasNext := params.Page < totalPages
	hasPrevious := params.Page > 1

	// Build filters applied response
	filtersApplied := FiltersApplied{
		Symbols:    params.Symbols,
		TradeTypes: params.TradeTypes,
		Exchanges:  params.Exchanges,
		Brokers:    params.Brokers,
		Currencies: params.Currencies,
	}

	// Add timeframe if dates were provided
	if params.StartDate != nil || params.EndDate != nil {
		timeframe := ""
		if params.StartDate != nil && params.EndDate != nil {
			timeframe = fmt.Sprintf("%s,%s",
				params.StartDate.Format("2006-01-02"),
				params.EndDate.Format("2006-01-02"))
		} else if params.StartDate != nil {
			timeframe = params.StartDate.Format("2006-01-02")
		} else if params.EndDate != nil {
			timeframe = params.EndDate.Format("2006-01-02")
		}
		filtersApplied.Timeframe = &timeframe
	}

	c.JSON(http.StatusOK, GetTransactionsResponse{
		Success: true,
		Message: "Transactions retrieved successfully",
		Data: &GetTransactionsData{
			Transactions: responseTransactions,
			Pagination: types.PaginationData{
				Page:         params.Page,
				PageSize:     params.PageSize,
				TotalRecords: int(totalCount),
				TotalPages:   totalPages,
				HasNext:      hasNext,
				HasPrevious:  hasPrevious,
			},
			FiltersApplied: filtersApplied,
		},
	})
}

// UpdateTransaction handles PUT /transaction-history/:id
func (h *TransactionsHandler) UpdateTransaction(c *gin.Context) {
	idParam := c.Param("id")
	if idParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Transaction ID is required"})
		return
	}

	transactionID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid transaction ID format"})
		return
	}

	var req TransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid request format", "errors": map[string][]string{"json": {"Invalid JSON format"}}})
		return
	}

	if err := validateTransaction(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Validation failed", "errors": map[string][]string{"validation": {err.Error()}}})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Unauthorized", "errors": map[string][]string{"auth": {"User not authenticated"}}})
		return
	}
	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Invalid user ID in context", "errors": map[string][]string{"auth": {"Invalid user ID in context"}}})
		return
	}

	updated, err := h.transactionService.UpdateTransaction(
		userUUID,
		transactionID,
		req.Symbol,
		req.Exchange,
		req.Broker,
		req.Currency,
		req.TradeDate,
		string(req.TradeType),
		req.Quantity,
		req.Price,
		req.Amount,
		req.UserNotes,
	)
	if err != nil {
		switch err.Error() {
		case "not_found":
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Transaction does not exist"})
			return
		case "forbidden":
			c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Not the owner"})
			return
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to update transaction"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Transaction updated successfully",
		"data":    gin.H{"transaction": modelToTransactionData(*updated)},
	})
}

// validateTransaction validates a single transaction request
func validateTransaction(transaction TransactionRequest) error {
	// Validate symbol length
	if len(transaction.Symbol) == 0 || len(transaction.Symbol) > 10 {
		return fmt.Errorf("symbol must be between 1 and 10 characters")
	}

	// Validate symbol format (alphanumeric uppercase, allow dot for e.g. BRK.B)
	if !utils.SymbolRegex.MatchString(transaction.Symbol) {
		return fmt.Errorf("symbol must contain only uppercase letters, numbers, and optionally a single dot (e.g. BRK.B)")
	}

	// Validate currency
	if len(transaction.Currency) != 3 {
		return fmt.Errorf("currency must be a 3-letter ISO code")
	}

	// Validate trade type
	validTradeTypes := map[types.TradeType]bool{
		types.TradeTypeBuy:      true,
		types.TradeTypeSell:     true,
		types.TradeTypeDividend: true,
	}
	if !validTradeTypes[transaction.TradeType] {
		return fmt.Errorf("trade_type must be one of: Buy, Sell, Dividends")
	}

	// Validate quantities and amounts
	if transaction.TradeType != types.TradeTypeDividend {
		if transaction.Quantity <= 0 {
			return fmt.Errorf("quantity must be positive")
		}
		if transaction.Price <= 0 {
			return fmt.Errorf("price must be positive")
		}
		if transaction.Amount <= 0 {
			return fmt.Errorf("amount must be positive")
		}

		// Validate trade amount calculation (with tolerance for rounding)
		expectedAmount := transaction.Quantity * transaction.Price
		tolerance := 0.1
		if utils.Abs(transaction.Amount-expectedAmount) > tolerance {
			return fmt.Errorf("amount does not match quantity × price calculation")
		}
	}

	// Validate date format and range
	tradeDate, err := time.Parse("2006-01-02", transaction.TradeDate)
	if err != nil {
		return fmt.Errorf("trade_date must be in YYYY-MM-DD format")
	}

	// Check date is not in the future
	if tradeDate.After(time.Now()) {
		return fmt.Errorf("trade_date cannot be in the future")
	}

	// Check date is not more than 30 years in the past
	thirtyYearsAgo := time.Now().AddDate(-30, 0, 0)
	if tradeDate.Before(thirtyYearsAgo) {
		return fmt.Errorf("trade_date cannot be more than 30 years in the past")
	}

	// Validate user notes length
	if len(transaction.UserNotes) > 1000 {
		return fmt.Errorf("user_notes cannot exceed 1000 characters")
	}

	return nil
}

// parseTransactionQueryParams parses and validates query parameters
func parseTransactionQueryParams(c *gin.Context) (params TransactionQueryParams, validationErrors map[string][]string) {
	validationErrors = make(map[string][]string)

	// Parse page
	page, err := utils.ParseUint(c.Query("page"), 1)
	if err != nil || page < 1 {
		validationErrors["page"] = []string{"Must be a positive integer"}
	} else {
		params.Page = int(page)
	}

	// Parse page_size
	pageSize, err := utils.ParseUint(c.Query("page_size"), 100)
	if err != nil || pageSize < 1 || pageSize > 1000 {
		validationErrors["page_size"] = []string{"Must be between 1 and 1000"}
	} else {
		params.PageSize = int(pageSize)
	}

	// Parse symbol (support multiple comma-separated symbols)
	if symbolParam := c.Query("symbol"); symbolParam != "" {
		symbols := strings.Split(symbolParam, ",")
		var validSymbols []string

		for _, symbol := range symbols {
			symbol = strings.TrimSpace(symbol)
			if !utils.SymbolRegex.MatchString(symbol) {
				validationErrors["symbol"] = []string{"Each symbol must be alphanumeric uppercase, 1-10 characters"}
				break
			} else {
				validSymbols = append(validSymbols, symbol)
			}
		}

		if len(validSymbols) > 0 && len(validationErrors) == 0 {
			params.Symbols = validSymbols
		}
	}

	// Parse types (comma-separated)
	if typesParam := c.Query("trade_type"); typesParam != "" {
		typeList := strings.Split(typesParam, ",")
		var validTypes []string

		for _, tradeType := range typeList {
			tradeType = strings.TrimSpace(tradeType)
			if _, ok := utils.TradeTypeFromString(tradeType); !ok {
				validationErrors["trade_type"] = []string{"Must be one of: Buy, Sell, Dividends (comma-separated for multiple)"}
				break
			} else {
				validTypes = append(validTypes, tradeType)
			}
		}

		if len(validTypes) > 0 && len(validationErrors) == 0 {
			params.TradeTypes = validTypes
		}
	}

	// Parse exchanges (comma-separated)
	if exchangesParam := c.Query("exchange"); exchangesParam != "" {
		exchanges := strings.Split(exchangesParam, ",")
		var validExchanges []string

		for _, exchange := range exchanges {
			exchange = strings.TrimSpace(exchange)
			if exchange != "" {
				validExchanges = append(validExchanges, exchange)
			}
		}

		if len(validExchanges) > 0 {
			params.Exchanges = validExchanges
		}
	}

	// Parse brokers (comma-separated)
	if brokersParam := c.Query("broker"); brokersParam != "" {
		brokers := strings.Split(brokersParam, ",")
		var validBrokers []string

		for _, broker := range brokers {
			broker = strings.TrimSpace(broker)
			if broker != "" {
				validBrokers = append(validBrokers, broker)
			}
		}

		if len(validBrokers) > 0 {
			params.Brokers = validBrokers
		}
	}

	// Parse currencies (comma-separated)
	if currenciesParam := c.Query("currency"); currenciesParam != "" {
		currencies := strings.Split(currenciesParam, ",")
		var validCurrencies []string

		for _, currency := range currencies {
			currency = strings.TrimSpace(currency)
			if !utils.CurrencyRegex.MatchString(currency) {
				validationErrors["currency"] = []string{"Must be valid 3-letter ISO currency codes (comma-separated for multiple)"}
				break
			} else {
				validCurrencies = append(validCurrencies, currency)
			}
		}

		if len(validCurrencies) > 0 && len(validationErrors) == 0 {
			params.Currencies = validCurrencies
		}
	}

	// Parse timeframe
	if timeframe := c.Query("timeframe"); timeframe != "" {
		if err := parseTimeframe(timeframe, &params); err != nil {
			validationErrors["timeframe"] = []string{err.Error()}
		}
	}

	// Parse sort_by
	params.SortBy = c.Query("sort_by")
	if params.SortBy == "" {
		params.SortBy = "transaction_date"
	} else {
		validSortFields := []string{"transaction_date", "symbol", "price", "quantity", "trade_amount"}
		valid := false
		for _, field := range validSortFields {
			if params.SortBy == field {
				valid = true
				break
			}
		}
		if !valid {
			validationErrors["sort_by"] = []string{"Must be one of: transaction_date, symbol, price, quantity, trade_amount"}
		}
	}

	// Parse sort_order
	params.SortOrder = c.Query("sort_order")
	if params.SortOrder == "" {
		params.SortOrder = "desc"
	} else if params.SortOrder != "asc" && params.SortOrder != "desc" {
		validationErrors["sort_order"] = []string{"Must be 'asc' or 'desc'"}
	}

	return params, validationErrors
}

// parseTimeframe parses timeframe query parameter (YYYY-MM-DD or YYYY-MM-DD,YYYY-MM-DD)
func parseTimeframe(timeframe string, params *TransactionQueryParams) error {
	// Check if it's a date range (contains comma)
	if len(timeframe) > 10 && timeframe[10] == ',' {
		// Date range: from,to
		parts := strings.Split(timeframe, ",")
		if len(parts) != 2 {
			return fmt.Errorf("invalid date range format, expected YYYY-MM-DD,YYYY-MM-DD")
		}

		startDate, err := time.Parse("2006-01-02", parts[0])
		if err != nil {
			return fmt.Errorf("invalid start date format, expected YYYY-MM-DD")
		}

		endDate, err := time.Parse("2006-01-02", parts[1])
		if err != nil {
			return fmt.Errorf("invalid end date format, expected YYYY-MM-DD")
		}

		if startDate.After(endDate) {
			return fmt.Errorf("start date must be before or equal to end date")
		}

		params.StartDate = &startDate
		params.EndDate = &endDate
	} else {
		// Single date
		singleDate, err := time.Parse("2006-01-02", timeframe)
		if err != nil {
			return fmt.Errorf("invalid date format, expected YYYY-MM-DD")
		}
		params.StartDate = &singleDate
		params.EndDate = &singleDate
	}

	return nil
}

// DeleteTransactionRequest represents the request structure for batch deleting transactions
type DeleteTransactionRequest struct {
	IDs []string `json:"ids" binding:"required,min=1"`
}

// DeleteTransactionResponse represents the response for deleting transactions
type DeleteTransactionResponse struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message"`
	Data    *DeleteTransactionData `json:"data,omitempty"`
	Errors  map[string][]string    `json:"errors,omitempty"`
}

// DeleteTransactionData represents the data part of delete transaction response
type DeleteTransactionData struct {
	DeletedIDs []string `json:"deleted_ids"`
}

// DeleteTransaction handles DELETE /transaction-history/:id
func (h *TransactionsHandler) DeleteTransaction(c *gin.Context) {
	idParam := c.Param("id")
	if idParam == "" {
		c.JSON(http.StatusBadRequest, DeleteTransactionResponse{
			Success: false,
			Message: "Transaction ID is required",
			Errors:  map[string][]string{"id": {"Transaction ID is required"}},
		})
		return
	}

	transactionID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, DeleteTransactionResponse{
			Success: false,
			Message: "Invalid transaction ID format",
			Errors:  map[string][]string{"id": {"Invalid transaction ID format"}},
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, DeleteTransactionResponse{
			Success: false,
			Message: "Unauthorized",
			Errors:  map[string][]string{"auth": {"User not authenticated"}},
		})
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, DeleteTransactionResponse{
			Success: false,
			Message: "Invalid user ID in context",
			Errors:  map[string][]string{"auth": {"Invalid user ID in context"}},
		})
		return
	}

	err = h.transactionService.DeleteTransaction(userUUID, transactionID)
	if err != nil {
		switch err.Error() {
		case "not_found":
			c.JSON(http.StatusNotFound, DeleteTransactionResponse{
				Success: false,
				Message: "Transaction does not exist",
				Errors:  map[string][]string{"id": {"Transaction does not exist"}},
			})
			return
		case "forbidden":
			c.JSON(http.StatusForbidden, DeleteTransactionResponse{
				Success: false,
				Message: "Not the owner",
				Errors:  map[string][]string{"auth": {"Not the owner"}},
			})
			return
		default:
			c.JSON(http.StatusInternalServerError, DeleteTransactionResponse{
				Success: false,
				Message: "Failed to delete transaction",
				Errors:  map[string][]string{"server": {"Failed to delete transaction"}},
			})
			return
		}
	}

	c.JSON(http.StatusOK, DeleteTransactionResponse{
		Success: true,
		Message: "Transaction deleted successfully",
		Data:    &DeleteTransactionData{DeletedIDs: []string{idParam}},
	})
}

// DeleteTransactions handles DELETE /transaction-history with batch deletion
func (h *TransactionsHandler) DeleteTransactions(c *gin.Context) {
	// Check if batch deletion is requested via query parameters
	idsParam := c.Query("ids")
	if idsParam != "" {
		// Handle batch deletion via query parameters
		h.handleBatchDeletionByQuery(c, idsParam)
		return
	}

	// Handle batch deletion via request body
	var req DeleteTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, DeleteTransactionResponse{
			Success: false,
			Message: "Invalid request format",
			Errors:  map[string][]string{"json": {"Invalid JSON format"}},
		})
		return
	}

	h.handleBatchDeletionByBody(c, req.IDs)
}

// handleBatchDeletionByQuery handles batch deletion via query parameters
func (h *TransactionsHandler) handleBatchDeletionByQuery(c *gin.Context, idsParam string) {
	idStrings := strings.Split(idsParam, ",")
	h.handleBatchDeletionByBody(c, idStrings)
}

// handleBatchDeletionByBody handles batch deletion logic
func (h *TransactionsHandler) handleBatchDeletionByBody(c *gin.Context, idStrings []string) {
	if len(idStrings) == 0 {
		c.JSON(http.StatusBadRequest, DeleteTransactionResponse{
			Success: false,
			Message: "At least one transaction ID is required",
			Errors:  map[string][]string{"ids": {"At least one transaction ID is required"}},
		})
		return
	}

	// Parse UUIDs
	var transactionIDs []uuid.UUID
	var validationErrors []string

	for _, idStr := range idStrings {
		idStr = strings.TrimSpace(idStr)
		if idStr == "" {
			continue
		}

		transactionID, err := uuid.Parse(idStr)
		if err != nil {
			validationErrors = append(validationErrors, fmt.Sprintf("Invalid transaction ID format: %s", idStr))
			continue
		}
		transactionIDs = append(transactionIDs, transactionID)
	}

	if len(validationErrors) > 0 {
		c.JSON(http.StatusBadRequest, DeleteTransactionResponse{
			Success: false,
			Message: "Validation failed",
			Errors:  map[string][]string{"ids": validationErrors},
		})
		return
	}

	if len(transactionIDs) == 0 {
		c.JSON(http.StatusBadRequest, DeleteTransactionResponse{
			Success: false,
			Message: "No valid transaction IDs provided",
			Errors:  map[string][]string{"ids": {"No valid transaction IDs provided"}},
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, DeleteTransactionResponse{
			Success: false,
			Message: "Unauthorized",
			Errors:  map[string][]string{"auth": {"User not authenticated"}},
		})
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, DeleteTransactionResponse{
			Success: false,
			Message: "Invalid user ID in context",
			Errors:  map[string][]string{"auth": {"Invalid user ID in context"}},
		})
		return
	}

	deletedIDs, err := h.transactionService.DeleteTransactions(userUUID, transactionIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, DeleteTransactionResponse{
			Success: false,
			Message: "Failed to delete transactions",
			Errors:  map[string][]string{"server": {"Failed to delete transactions"}},
		})
		return
	}

	// Convert UUIDs back to strings
	deletedIDStrings := make([]string, len(deletedIDs))
	for i, id := range deletedIDs {
		deletedIDStrings[i] = id.String()
	}

	var message string
	if len(deletedIDStrings) == 1 {
		message = "Transaction deleted successfully"
	} else {
		message = fmt.Sprintf("%d transactions deleted successfully", len(deletedIDStrings))
	}

	c.JSON(http.StatusOK, DeleteTransactionResponse{
		Success: true,
		Message: message,
		Data:    &DeleteTransactionData{DeletedIDs: deletedIDStrings},
	})
}
