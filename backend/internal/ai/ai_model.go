package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	"github.com/transaction-tracker/backend/internal/constants"
	"github.com/transaction-tracker/backend/internal/prompts"
	"github.com/transaction-tracker/backend/internal/types"
	"google.golang.org/api/option"
)

// ModelType represents different AI model providers
type ModelType string

const (
	ModelTypeGemini ModelType = "gemini"
)

type AIModelClient struct {
	modelType ModelType
	config    *Config

	// Provider-specific clients (only one will be active at a time)
	geminiClient *genai.Client
	geminiModel  *genai.GenerativeModel
}

// NewAIModelClient creates a new AI model client based on the configuration
func NewAIModelClient(config *Config) (*AIModelClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("API key is required for AI model client")
	}

	// Determine model type from config
	modelType := determineModelType(config.Model)

	client := &AIModelClient{
		modelType: modelType,
		config:    config,
	}

	// Initialize the appropriate provider client
	switch modelType {
	case ModelTypeGemini:
		if err := client.initGeminiClient(); err != nil {
			return nil, fmt.Errorf("failed to initialize Gemini client: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported model type: %s", modelType)
	}

	return client, nil
}

// determineModelType determines the AI provider based on the model name
func determineModelType(modelName string) ModelType {
	if modelName == "" {
		modelName = constants.DefaultAIModel
	}

	// Check for Gemini models
	switch modelName {
	case "gemini-2.0-flash", "gemini-1.5-pro", "gemini-1.5-flash":
		return ModelTypeGemini
	}

	// Default to Gemini for backward compatibility
	return ModelTypeGemini
}

// initGeminiClient initializes the Gemini client
func (c *AIModelClient) initGeminiClient() error {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(c.config.APIKey))
	if err != nil {
		return fmt.Errorf("failed to create Gemini client: %w", err)
	}

	// Set default model if not specified
	modelName := c.config.Model
	if modelName == "" {
		modelName = constants.DefaultAIModel
	}

	model := client.GenerativeModel(modelName)

	// Configure model settings for better JSON responses
	model.ResponseMIMEType = constants.MimeTypeJSON

	// Load system instruction from prompt file
	systemInstruction, err := prompts.LoadPrompt("system_instruction.txt")
	if err != nil {
		return fmt.Errorf("failed to load system instruction: %w", err)
	}

	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(systemInstruction)},
	}

	c.geminiClient = client
	c.geminiModel = model

	return nil
}

// ExtractTransactions processes a single image and extracts transaction data using the configured AI model
func (c *AIModelClient) ExtractTransactions(ctx context.Context, image types.FileInput) (*types.ExtractResponse, error) {
	// Check for mock responses based on filename (only in development environment and filename prefix "mock_")
	if c.config.Environment == "development" && strings.HasPrefix(image.Filename, "mock") {
		if mockResponse := c.getMockResponse(image.Filename); mockResponse != nil {
			return mockResponse, nil
		}
	}

	// Route to the appropriate implementation based on model type
	switch c.modelType {
	case ModelTypeGemini:
		return c.extractTransactionsGemini(ctx, image)
	default:
		return &types.ExtractResponse{
			Success: false,
			Message: fmt.Sprintf("Unsupported model type: %s", c.modelType),
		}, fmt.Errorf("unsupported model type: %s", c.modelType)
	}
}

// extractTransactionsGemini handles transaction extraction using Gemini models
func (c *AIModelClient) extractTransactionsGemini(ctx context.Context, image types.FileInput) (*types.ExtractResponse, error) {
	// Load the transaction extraction prompt
	prompt, promptErr := prompts.LoadPrompt("transaction_extraction.txt")
	if promptErr != nil {
		return &types.ExtractResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to load extraction prompt: %v", promptErr),
		}, fmt.Errorf("failed to load extraction prompt: %w", promptErr)
	}

	// Prepare the content parts for the request
	parts := []genai.Part{genai.Text(prompt)}

	// Add the image to the request
	imageData, readErr := io.ReadAll(image.Data)
	if readErr != nil {
		return &types.ExtractResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to read image %s: %v", image.Filename, readErr),
		}, fmt.Errorf("failed to read image %s: %w", image.Filename, readErr)
	}

	parts = append(parts, genai.ImageData(image.MimeType, imageData))

	// Set timeout if specified
	if c.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(c.config.Timeout)*time.Second)
		defer cancel()
	}

	// Generate content with retry logic
	var resp *genai.GenerateContentResponse
	var err error

	maxRetry := c.config.MaxRetry
	if maxRetry <= 0 {
		maxRetry = constants.DefaultAIMaxRetry
	}

	for attempt := 0; attempt < maxRetry; attempt++ {
		resp, err = c.geminiModel.GenerateContent(ctx, parts...)
		if err == nil {
			break
		}

		if attempt < maxRetry-1 {
			log.Printf("Attempt %d failed, retrying: %v", attempt+1, err)
			time.Sleep(time.Duration(attempt+1) * time.Second)
		}
	}

	if err != nil {
		return &types.ExtractResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to generate content after %d attempts: %v", maxRetry, err),
		}, err
	}

	// Parse the response
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return &types.ExtractResponse{
			Success: false,
			Message: "No response received from AI model",
		}, nil
	}

	// Extract the text response
	responseText := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])

	return c.parseTransactionResponse(responseText, image.Filename)
}

// parseTransactionResponse parses the AI response into transaction data (generic for all models)
func (c *AIModelClient) parseTransactionResponse(responseText string, filename string) (*types.ExtractResponse, error) {
	// Parse JSON response
	var result struct {
		Transactions []types.TransactionData `json:"transactions"`
	}

	if err := json.Unmarshal([]byte(responseText), &result); err != nil {
		log.Printf("Failed to parse AI response as JSON: %v\nResponse: %s", err, responseText)
		return &types.ExtractResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to parse AI response: %v", err),
		}, nil
	}

	return &types.ExtractResponse{
		Data: &types.ExtractResponseData{
			Transactions:     result.Transactions,
			TransactionCount: len(result.Transactions),
			FileName:         filename,
		},
		Success: true,
		Message: constants.MsgTransactionsExtracted,
	}, nil
}

// Health checks if the AI model client is working properly
func (c *AIModelClient) Health(ctx context.Context) error {
	switch c.modelType {
	case ModelTypeGemini:
		return c.healthCheckGemini(ctx)
	default:
		return fmt.Errorf("health check not implemented for model type: %s", c.modelType)
	}
}

// healthCheckGemini performs a health check for Gemini models
func (c *AIModelClient) healthCheckGemini(ctx context.Context) error {
	// Simple health check by making a minimal request
	parts := []genai.Part{genai.Text("Health check - please respond with 'OK'")}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := c.geminiModel.GenerateContent(ctx, parts...)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	return nil
}

// Close closes the client and cleans up resources
func (c *AIModelClient) Close() error {
	switch c.modelType {
	case ModelTypeGemini:
		if c.geminiClient != nil {
			return c.geminiClient.Close()
		}
	}
	// Future model cleanup logic can be added here
	return nil
}

// GetModelType returns the current model type
func (c *AIModelClient) GetModelType() ModelType {
	return c.modelType
}

// GetModelName returns the configured model name
func (c *AIModelClient) GetModelName() string {
	return c.config.Model
}

// getMockResponse returns mock responses based on filename for development/testing
func (c *AIModelClient) getMockResponse(filename string) *types.ExtractResponse {
	// Get filename without extension for flexible matching
	fileNameWithoutExt := strings.TrimSuffix(filename, filepath.Ext(filename))

	// Helper to create mock transaction data
	createMockTransaction := func(index int) types.TransactionData {
		symbols := []string{"AAPL", "GOOGL", "MSFT", "TSLA", "AMZN", "NVDA", "META", "NFLX"}
		tradeTypes := []types.TradeType{types.TradeTypeBuy, types.TradeTypeSell, types.TradeTypeDividend}

		quantity := float64(rand.Intn(100) + 1)
		price := float64(rand.Intn(45000)+5000) / 100.0 // $50-500

		return types.TransactionData{
			Symbol:          symbols[index%len(symbols)],
			TradeType:       tradeTypes[index%len(tradeTypes)],
			Quantity:        quantity,
			Price:           price,
			Amount:          quantity * price,
			Currency:        "USD",
			Broker:          "Mock Broker",
			Account:         "Mock Account",
			Exchange:        "NASDAQ",
			TransactionDate: time.Now().AddDate(0, 0, -rand.Intn(30)).Format("2006-01-02"),
			UserNotes:       fmt.Sprintf("Mock transaction %d", index+1),
		}
	}

	switch fileNameWithoutExt {
	case "mock_file":
		// Normal case: 10 transactions with 2-3 second delay
		time.Sleep(time.Duration(2000+rand.Intn(1000)) * time.Millisecond)

		transactions := make([]types.TransactionData, 10)
		for i := range transactions {
			transactions[i] = createMockTransaction(i)
		}

		return &types.ExtractResponse{
			Success: true,
			Message: constants.MsgTransactionsExtracted,
			Data: &types.ExtractResponseData{
				Transactions:     transactions,
				TransactionCount: 10,
				FileName:         filename,
			},
		}

	case "mock_file_error":
		// Error case: 1-2 second delay then error
		time.Sleep(time.Duration(1000+rand.Intn(1000)) * time.Millisecond)

		return &types.ExtractResponse{
			Success: false,
			Message: "Processing failed: Unable to extract transactions from image. The image quality may be too low or the format is not supported.",
		}

	case "mock_file_empty":
		// Empty case: 1-2 second delay, 0 transactions
		time.Sleep(time.Duration(1000+rand.Intn(1000)) * time.Millisecond)

		return &types.ExtractResponse{
			Success: true,
			Message: "No transactions found in the image",
			Data: &types.ExtractResponseData{
				Transactions:     []types.TransactionData{},
				TransactionCount: 0,
				FileName:         filename,
			},
		}

	case "mock_file_pending":
		// Long processing case: 10-15 second delay
		time.Sleep(time.Duration(10000+rand.Intn(5000)) * time.Millisecond)

		transactions := make([]types.TransactionData, 5)
		for i := range transactions {
			transactions[i] = createMockTransaction(i)
		}

		return &types.ExtractResponse{
			Success: true,
			Message: "Transactions extracted successfully after long processing",
			Data: &types.ExtractResponseData{
				Transactions:     transactions,
				TransactionCount: 5,
				FileName:         filename,
			},
		}
	}

	// Return nil to indicate we should use the real AI
	return nil
}
