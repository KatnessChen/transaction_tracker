package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/transaction-tracker/backend/config"
	"github.com/transaction-tracker/backend/internal/constants"
	"github.com/transaction-tracker/backend/internal/types"
)

// AIClient defines the interface for interacting with AI services
type AIClient interface {
	ExtractTransactions(ctx context.Context, image types.FileInput) (*types.ExtractResponse, error)
}

// ExtractHandler handles extraction endpoints and dependencies
type ExtractTransactionHandler struct {
	aiClient AIClient
	cfg      *config.Config
}

// NewExtractTransactionsHandler creates a new ExtractHandler
func NewExtractTransactionsHandler(cfg *config.Config, aiClient AIClient) *ExtractTransactionHandler {
	return &ExtractTransactionHandler{cfg: cfg, aiClient: aiClient}
}

// ExtractTransactionsHandler handles the image upload and transaction extraction
func (h *ExtractTransactionHandler) ExtractTransactions(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ExtractResponse{
			Success: false,
			Message: "Failed to parse multipart form: " + err.Error(),
		})
		return
	}
	file := form.File["file"]

	if len(file) == 0 {
		c.JSON(http.StatusBadRequest, types.ExtractResponse{
			Success: false,
			Message: "No file uploaded. Please upload a file under the 'file' field.",
		})
		return
	}

	fileHeader := file[0]
	src, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.ExtractResponse{
			Success: false,
			Message: "Failed to open uploaded file " + fileHeader.Filename + ": " + err.Error(),
		})
		return
	}
	defer src.Close()

	imageInput := types.FileInput{
		Data:     src,
		Filename: fileHeader.Filename,
		MimeType: fileHeader.Header.Get(constants.ContentTypeHeader),
	}

	extractResp, err := h.aiClient.ExtractTransactions(c.Request.Context(), imageInput)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.ExtractResponse{
			Success: false,
			Message: "Failed to extract transactions: " + err.Error(),
		})
		return
	}

	if !extractResp.Success {
		c.JSON(http.StatusInternalServerError, extractResp)
		return
	}

	c.JSON(http.StatusOK, extractResp)
}
