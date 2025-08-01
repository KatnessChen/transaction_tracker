package repositories

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/transaction-tracker/backend/internal/models"
	"gorm.io/gorm"
)

// TransactionRepository handles transaction database operations
type TransactionRepository struct {
	db *gorm.DB
}

// NewTransactionRepository creates a new transaction repository
func NewTransactionRepository(db *gorm.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

// Create creates a single transaction
func (r *TransactionRepository) Create(transaction *models.Transaction) error {
	return r.db.Create(transaction).Error
}

// CreateMany creates multiple transactions in a single database transaction
func (r *TransactionRepository) CreateMany(transactions []models.Transaction) ([]models.Transaction, error) {
	// Start a database transaction
	tx := r.db.Begin()
	if tx.Error != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}

	// Ensure rollback on any error
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var createdTransactions []models.Transaction

	for i, transaction := range transactions {
		// Create transaction
		if err := tx.Create(&transaction).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to create transaction %d: %w", i+1, err)
		}
		createdTransactions = append(createdTransactions, transaction)
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return createdTransactions, nil
}

// GetByID retrieves a transaction by transaction_id (UUID)
func (r *TransactionRepository) GetByID(id uuid.UUID) (*models.Transaction, error) {
	var transaction models.Transaction
	err := r.db.Preload("User").Where("transaction_id = ?", id).First(&transaction).Error
	if err != nil {
		return nil, err
	}
	return &transaction, nil
}

// GetByIDAndUserID retrieves a transaction by transaction_id and user_id (UUID)
func (r *TransactionRepository) GetByIDAndUserID(id uuid.UUID, userID uuid.UUID) (*models.Transaction, error) {
	var transaction models.Transaction
	err := r.db.Where("transaction_id = ? AND user_id = ?", id, userID).First(&transaction).Error
	if err != nil {
		return nil, err
	}
	return &transaction, nil
}

// GetByUserID retrieves all transactions for a user by user_id (UUID)
func (r *TransactionRepository) GetByUserID(userID uuid.UUID) ([]models.Transaction, error) {
	var transactions []models.Transaction
	err := r.db.Where("user_id = ?", userID).Preload("User").Find(&transactions).Error
	return transactions, err
}

// UpdateByID updates a transaction by transaction_id (UUID)
func (r *TransactionRepository) UpdateByID(id uuid.UUID, updates map[string]interface{}) error {
	return r.db.Model(&models.Transaction{}).Where("transaction_id = ?", id).Updates(updates).Error
}

// DeleteByIDAndUserID soft deletes a transaction by transaction_id and user_id (UUID)
func (r *TransactionRepository) DeleteByIDAndUserID(id uuid.UUID, userID uuid.UUID) error {
	return r.db.Where("transaction_id = ? AND user_id = ?", id, userID).Delete(&models.Transaction{}).Error
}

// DeleteByIDsAndUserID soft deletes multiple transactions by transaction_ids and user_id (UUID)
func (r *TransactionRepository) DeleteByIDsAndUserID(ids []uuid.UUID, userID uuid.UUID) ([]uuid.UUID, error) {
	var deletedIDs []uuid.UUID

	// Start a database transaction
	tx := r.db.Begin()
	if tx.Error != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}

	// Ensure rollback on any error
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	for _, id := range ids {
		// Check if transaction exists and belongs to user
		var transaction models.Transaction
		if err := tx.Where("transaction_id = ? AND user_id = ?", id, userID).First(&transaction).Error; err != nil {
			// Skip non-existent or unauthorized transactions
			continue
		}

		// Delete the transaction
		if err := tx.Where("transaction_id = ? AND user_id = ?", id, userID).Delete(&models.Transaction{}).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to delete transaction %s: %w", id.String(), err)
		}

		deletedIDs = append(deletedIDs, id)
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return deletedIDs, nil
}

// GetWithFilters retrieves transactions with advanced filtering
func (r *TransactionRepository) GetWithFilters(userID *uuid.UUID, symbols []string, types []string, exchanges []string, brokers []string, currencies []string,
	startDate *time.Time, endDate *time.Time, minAmount *float64, maxAmount *float64,
	orderBy string, orderDirection string, limit int, offset int) ([]models.Transaction, error) {

	var transactions []models.Transaction
	query := r.db.Model(&models.Transaction{})

	// Apply filters
	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}
	if len(symbols) > 0 {
		query = query.Where("symbol IN ?", symbols)
	}
	if len(types) > 0 {
		query = query.Where("trade_type IN ?", types)
	}
	if len(exchanges) > 0 {
		query = query.Where("exchange IN ?", exchanges)
	}
	if len(brokers) > 0 {
		query = query.Where("broker IN ?", brokers)
	}
	if len(currencies) > 0 {
		query = query.Where("currency IN ?", currencies)
	}
	if startDate != nil {
		query = query.Where("transaction_date >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("transaction_date <= ?", *endDate)
	}
	if minAmount != nil {
		query = query.Where("amount >= ?", *minAmount)
	}
	if maxAmount != nil {
		query = query.Where("amount <= ?", *maxAmount)
	}

	// Apply ordering
	if orderBy == "" {
		orderBy = "transaction_date"
	}
	if orderDirection == "" {
		orderDirection = "DESC"
	}
	query = query.Order(fmt.Sprintf("%s %s", orderBy, orderDirection))

	// Apply pagination
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	// Preload user data
	query = query.Preload("User")

	if err := query.Find(&transactions).Error; err != nil {
		return nil, fmt.Errorf("failed to get filtered transactions: %w", err)
	}

	return transactions, nil
}

// CountWithFilters returns the count of transactions based on filters
func (r *TransactionRepository) CountWithFilters(userID *uuid.UUID, symbols []string, types []string, exchanges []string, brokers []string, currencies []string,
	startDate *time.Time, endDate *time.Time, minAmount *float64, maxAmount *float64) (int64, error) {

	var count int64
	query := r.db.Model(&models.Transaction{})

	// Apply filters
	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}
	if len(symbols) > 0 {
		query = query.Where("symbol IN ?", symbols)
	}
	if len(types) > 0 {
		query = query.Where("trade_type IN ?", types)
	}
	if len(exchanges) > 0 {
		query = query.Where("exchange IN ?", exchanges)
	}
	if len(brokers) > 0 {
		query = query.Where("broker IN ?", brokers)
	}
	if len(currencies) > 0 {
		query = query.Where("currency IN ?", currencies)
	}
	if startDate != nil {
		query = query.Where("transaction_date >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("transaction_date <= ?", *endDate)
	}
	if minAmount != nil {
		query = query.Where("amount >= ?", *minAmount)
	}
	if maxAmount != nil {
		query = query.Where("amount <= ?", *maxAmount)
	}

	if err := query.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count filtered transactions: %w", err)
	}

	return count, nil
}

// GetPortfolioSummaryByUserID returns portfolio summary for a user
func (r *TransactionRepository) GetPortfolioSummaryByUserID(userID uuid.UUID) (map[string]interface{}, error) {
	var result struct {
		TotalTransactions int64   `json:"total_transactions"`
		TotalBuyAmount    float64 `json:"total_buy_amount"`
		TotalSellAmount   float64 `json:"total_sell_amount"`
		UniqueSymbols     int64   `json:"unique_symbols"`
	}

	// Count total transactions
	if err := r.db.Model(&models.Transaction{}).Where("user_id = ?", userID).Count(&result.TotalTransactions).Error; err != nil {
		return nil, fmt.Errorf("failed to count transactions: %w", err)
	}

	// Sum buy amounts
	if err := r.db.Model(&models.Transaction{}).Where("user_id = ? AND trade_type = ?", userID, "Buy").Select("COALESCE(SUM(amount), 0)").Scan(&result.TotalBuyAmount).Error; err != nil {
		return nil, fmt.Errorf("failed to sum buy amounts: %w", err)
	}

	// Sum sell amounts
	if err := r.db.Model(&models.Transaction{}).Where("user_id = ? AND trade_type = ?", userID, "Sell").Select("COALESCE(SUM(amount), 0)").Scan(&result.TotalSellAmount).Error; err != nil {
		return nil, fmt.Errorf("failed to sum sell amounts: %w", err)
	}

	// Count unique symbols
	if err := r.db.Model(&models.Transaction{}).Where("user_id = ?", userID).Distinct("symbol").Count(&result.UniqueSymbols).Error; err != nil {
		return nil, fmt.Errorf("failed to count unique symbols: %w", err)
	}

	return map[string]interface{}{
		"total_transactions": result.TotalTransactions,
		"total_buy_amount":   result.TotalBuyAmount,
		"total_sell_amount":  result.TotalSellAmount,
		"unique_symbols":     result.UniqueSymbols,
		"net_amount":         result.TotalSellAmount - result.TotalBuyAmount,
	}, nil
}

// GetSymbolHoldingsByUserID returns current holdings for a user grouped by symbol
func (r *TransactionRepository) GetSymbolHoldingsByUserID(userID uuid.UUID) ([]map[string]interface{}, error) {
	var holdings []struct {
		Symbol       string  `json:"symbol"`
		TotalBought  float64 `json:"total_bought"`
		TotalSold    float64 `json:"total_sold"`
		NetQuantity  float64 `json:"net_quantity"`
		AvgBuyPrice  float64 `json:"avg_buy_price"`
		AvgSellPrice float64 `json:"avg_sell_price"`
	}

	query := `
		SELECT
			symbol,
			COALESCE(SUM(CASE WHEN trade_type = 'Buy' THEN quantity ELSE 0 END), 0) as total_bought,
			COALESCE(SUM(CASE WHEN trade_type = 'Sell' THEN quantity ELSE 0 END), 0) as total_sold,
			COALESCE(SUM(CASE WHEN trade_type = 'Buy' THEN quantity WHEN trade_type = 'Sell' THEN -quantity ELSE 0 END), 0) as net_quantity,
			COALESCE(AVG(CASE WHEN trade_type = 'Buy' THEN price ELSE NULL END), 0) as avg_buy_price,
			COALESCE(AVG(CASE WHEN trade_type = 'Sell' THEN price ELSE NULL END), 0) as avg_sell_price
		FROM transactions
		WHERE user_id = ? AND deleted_at IS NULL
		GROUP BY symbol
		HAVING net_quantity != 0
		ORDER BY symbol
	`

	if err := r.db.Raw(query, userID).Scan(&holdings).Error; err != nil {
		return nil, fmt.Errorf("failed to get symbol holdings: %w", err)
	}

	result := make([]map[string]interface{}, len(holdings))
	for i, holding := range holdings {
		result[i] = map[string]interface{}{
			"symbol":         holding.Symbol,
			"total_bought":   holding.TotalBought,
			"total_sold":     holding.TotalSold,
			"net_quantity":   holding.NetQuantity,
			"avg_buy_price":  holding.AvgBuyPrice,
			"avg_sell_price": holding.AvgSellPrice,
		}
	}

	return result, nil
}

// GetByUserIDAndSymbol retrieves all transactions for a specific user and symbol
func (r *TransactionRepository) GetByUserIDAndSymbol(userID uuid.UUID, symbol string) ([]models.Transaction, error) {
	var transactions []models.Transaction
	err := r.db.Where("user_id = ? AND symbol = ?", userID, symbol).
		Order("transaction_date ASC").
		Find(&transactions).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions for user %s and symbol %s: %w", userID, symbol, err)
	}
	return transactions, nil
}
