package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/transaction-tracker/price_service/internal/config"
	"github.com/transaction-tracker/price_service/internal/models"
)

type Service struct {
	client     *redis.Client
	defaultTTL time.Duration
}

func NewService(cfg *config.Config) (*Service, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &Service{
		client:     rdb,
		defaultTTL: cfg.Cache.DefaultTTL,
	}, nil
}

// Current price cache methods
func (s *Service) GetCurrentPrice(ctx context.Context, symbol string) (*models.SymbolCurrentPrice, error) {
	key := fmt.Sprintf("price_service:current-price:%s", symbol)
	val, err := s.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Not found
		}
		return nil, err
	}

	var price models.SymbolCurrentPrice
	if err := json.Unmarshal([]byte(val), &price); err != nil {
		return nil, err
	}

	return &price, nil
}

func (s *Service) SetCurrentPrice(ctx context.Context, symbol string, price *models.SymbolCurrentPrice) error {
	key := fmt.Sprintf("price_service:current-price:%s", symbol)
	data, err := json.Marshal(price)
	if err != nil {
		return err
	}

	return s.client.Set(ctx, key, data, s.defaultTTL).Err()
}

// Historical price cache methods
func (s *Service) GetHistoricalPrice(ctx context.Context, symbol string, resolution models.Resolution) (*models.SymbolHistoricalPrice, error) {
	today := time.Now().Format("2006-01-02")
	key := fmt.Sprintf("price_service:historical-price:%s:%s:%s", symbol, string(resolution), today)
	val, err := s.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Not found
		}
		return nil, err
	}

	var price models.SymbolHistoricalPrice
	if err := json.Unmarshal([]byte(val), &price); err != nil {
		return nil, err
	}

	return &price, nil
}

func (s *Service) SetHistoricalPrice(ctx context.Context, symbol string, resolution models.Resolution, price *models.SymbolHistoricalPrice) error {
	today := time.Now().Format("2006-01-02")
	key := fmt.Sprintf("price_service:historical-price:%s:%s:%s", symbol, string(resolution), today)
	data, err := json.Marshal(price)
	if err != nil {
		return err
	}

	// Historical data uses 24 hour TTL
	historicalTTL := 24 * time.Hour
	return s.client.Set(ctx, key, data, historicalTTL).Err()
}

// Cache management methods
func (s *Service) InvalidateAll(ctx context.Context) error {
	// Delete all price-related keys
	currentPriceKeys, err := s.client.Keys(ctx, "price_service:current-price:*").Result()
	if err != nil {
		return err
	}

	historicalPriceKeys, err := s.client.Keys(ctx, "price_service:historical-price:*:*:*").Result()
	if err != nil {
		return err
	}

	allKeys := append(currentPriceKeys, historicalPriceKeys...)
	if len(allKeys) > 0 {
		return s.client.Del(ctx, allKeys...).Err()
	}

	return nil
}

func (s *Service) Close() error {
	return s.client.Close()
}
