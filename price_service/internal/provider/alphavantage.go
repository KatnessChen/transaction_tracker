package provider

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/transaction-tracker/price_service/internal/models"
)

//go:embed testdata/alphavantage-ibm-daily.json
var ibmDailyTestData []byte

// AlphaVantageProvider handles historical stock prices from Alpha Vantage API
type AlphaVantageProvider struct {
	APIKey  string
	BaseURL string
	client  *http.Client
}

func NewAlphaVantageProvider(apiKey string) *AlphaVantageProvider {
	return &AlphaVantageProvider{
		APIKey:  apiKey,
		BaseURL: "https://www.alphavantage.co/query",
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (a *AlphaVantageProvider) GetHistoricalPrices(ctx context.Context, symbol string, resolution models.Resolution) (*models.SymbolHistoricalPrice, error) {

	params := url.Values{}
	// Map our resolution to Alpha Vantage function
	switch resolution {
	case models.ResolutionDaily:
		params.Set("function", "TIME_SERIES_DAILY")
	case models.ResolutionWeekly:
		params.Set("function", "TIME_SERIES_WEEKLY")
	case models.ResolutionMonthly:
		params.Set("function", "TIME_SERIES_MONTHLY")
	default:
		return nil, fmt.Errorf("unsupported resolution: %s", resolution)
	}

	params.Set("symbol", symbol)
	params.Set("apikey", a.APIKey)
	params.Set("outputsize", "full") // Get full historical data

	var resp []byte
	var err error
	if strings.EqualFold(symbol, "IBM") && resolution == models.ResolutionDaily {
		// Read mock response for IBM daily from file to avoid rate limit
		resp, err = readIBMTestData()
		if err != nil {
			return nil, fmt.Errorf("failed to read IBM mock data: %w", err)
		}
	} else {
		resp, err = a.makeRequest(ctx, params)
		if err != nil {
			return nil, err
		}
	}

	// Parse the response based on resolution
	var timeSeries map[string]map[string]string
	var result map[string]interface{}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse Alpha Vantage response: %w", err)
	}

	// Extract time series data based on resolution
	switch resolution {
	case models.ResolutionDaily:
		if ts, ok := result["Time Series (Daily)"].(map[string]interface{}); ok {
			timeSeries = make(map[string]map[string]string)
			for date, data := range ts {
				if dataMap, ok := data.(map[string]interface{}); ok {
					convertedData := make(map[string]string)
					for k, v := range dataMap {
						if str, ok := v.(string); ok {
							convertedData[k] = str
						}
					}
					timeSeries[date] = convertedData
				}
			}
		}
	case models.ResolutionWeekly:
		if ts, ok := result["Weekly Time Series"].(map[string]interface{}); ok {
			timeSeries = make(map[string]map[string]string)
			for date, data := range ts {
				if dataMap, ok := data.(map[string]interface{}); ok {
					convertedData := make(map[string]string)
					for k, v := range dataMap {
						if str, ok := v.(string); ok {
							convertedData[k] = str
						}
					}
					timeSeries[date] = convertedData
				}
			}
		}
	case models.ResolutionMonthly:
		if ts, ok := result["Monthly Time Series"].(map[string]interface{}); ok {
			timeSeries = make(map[string]map[string]string)
			for date, data := range ts {
				if dataMap, ok := data.(map[string]interface{}); ok {
					convertedData := make(map[string]string)
					for k, v := range dataMap {
						if str, ok := v.(string); ok {
							convertedData[k] = str
						}
					}
					timeSeries[date] = convertedData
				}
			}
		}
	}

	var prices []models.ClosePrice
	for dateStr, data := range timeSeries {
		// Get closing price
		if closePrice, ok := data["4. close"]; ok {
			if price, err := strconv.ParseFloat(closePrice, 64); err == nil {
				prices = append(prices, models.ClosePrice{
					Date:  dateStr,
					Price: price,
				})
			}
		}
	}

	// Sort prices by date (newest to oldest)
	sort.Slice(prices, func(i, j int) bool {
		d1, err1 := time.Parse("2006-01-02", prices[i].Date)
		d2, err2 := time.Parse("2006-01-02", prices[j].Date)
		if err1 != nil || err2 != nil {
			// Fallback: compare as strings (descending)
			return prices[i].Date > prices[j].Date
		}
		return d1.After(d2)
	})

	return &models.SymbolHistoricalPrice{
		Symbol:           symbol,
		Resolution:       resolution,
		HistoricalPrices: prices,
	}, nil
}

func (a *AlphaVantageProvider) makeRequest(ctx context.Context, params url.Values) ([]byte, error) {
	reqURL := fmt.Sprintf("%s?%s", a.BaseURL, params.Encode())

	// Log the request URL for debugging (sanitize API key for security)
	sanitizedURL := strings.Replace(reqURL, a.APIKey, "[API_KEY]", -1)
	log.Printf("Alpha Vantage API Request: %s", sanitizedURL)

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("alpha vantage API error: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

// readIBMTestData loads the IBM daily mock JSON from the test data directory
func readIBMTestData() ([]byte, error) {
	return ibmDailyTestData, nil
}
