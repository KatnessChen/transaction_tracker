package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server    ServerConfig
	Redis     RedisConfig
	StockAPI  StockAPIConfig
	Cache     CacheConfig
	RateLimit RateLimitConfig
}

type ServerConfig struct {
	Port   string
	APIKey string
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type StockAPIConfig struct {
	Provider string
	APIKey   string
	BaseURL  string
}

type CacheConfig struct {
	DefaultTTL       time.Duration
	MaxSymbolsPerReq int
}

type RateLimitConfig struct {
	RequestsPerWindow int
	WindowDuration    time.Duration
}

func Load() (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load()

	config := &Config{
		Server: ServerConfig{
			Port:   getEnv("PORT", "8081"),
			APIKey: getEnv("API_KEY", ""),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		StockAPI: StockAPIConfig{
			Provider: getEnv("STOCK_API_PROVIDER", "alpha_vantage"),
			APIKey:   getEnv("STOCK_API_KEY", ""),
			BaseURL:  getEnv("STOCK_API_BASE_URL", ""),
		},
		Cache: CacheConfig{
			DefaultTTL:       time.Duration(getEnvAsInt("DEFAULT_TTL_MINUTES", 60)) * time.Minute,
			MaxSymbolsPerReq: getEnvAsInt("MAX_SYMBOLS_PER_REQUEST", 50),
		},
		RateLimit: RateLimitConfig{
			RequestsPerWindow: getEnvAsInt("RATE_LIMIT_REQUESTS", 100),
			WindowDuration:    time.Duration(getEnvAsInt("RATE_LIMIT_WINDOW_MINUTES", 1)) * time.Minute,
		},
	}

	return config, nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvAsInt(name string, fallback int) int {
	valueStr := getEnv(name, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return fallback
}
