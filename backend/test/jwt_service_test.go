package test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/transaction-tracker/backend/internal/utils"
	"gorm.io/gorm"

	"github.com/transaction-tracker/backend/config"
	"github.com/transaction-tracker/backend/internal/models"
	"github.com/transaction-tracker/backend/internal/repositories"
	"github.com/transaction-tracker/backend/internal/services"
)

// Test setup helper
func setupJWTServiceTest(t *testing.T) (*gorm.DB, *config.Config, services.JWTService, *models.User) {
	// Use shared MySQL test DB
	db := utils.SetupTestDB(t)

	// Create test config
	cfg := &config.Config{
		JWTSecret:          "test-secret-key-for-jwt-testing",
		JWTExpirationHours: 24,
	}

	// Create repositories and service
	jwtRepo := repositories.NewJWTRepository(db)
	jwtService := services.NewJWTService(cfg, jwtRepo)

	// Create test user
	testUser := &models.User{
		Username:  "jwt_test_user",
		Email:     "jwt.test@example.com",
		FirstName: "JWT",
		LastName:  "Testuser",
		IsActive:  true,
	}
	err := testUser.SetPassword("TestPass123!")
	require.NoError(t, err)

	err = db.Create(testUser).Error
	require.NoError(t, err)

	return db, cfg, jwtService, testUser
}

// Helper function to create test device info
func createTestDeviceInfo() services.DeviceInfo {
	return services.DeviceInfo{
		UserAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
		IPAddress: "192.168.1.100",
		Browser:   "Chrome",
		OS:        "macOS",
	}
}

// Test 1.1: Token Generation Tests
func TestJWTService_GenerateToken(t *testing.T) {
	db, cfg, jwtService, testUser := setupJWTServiceTest(t)
	deviceInfo := createTestDeviceInfo()

	// Test successful token generation
	tokenString, err := jwtService.GenerateToken(testUser, deviceInfo)
	assert.NoError(t, err)
	assert.NotEmpty(t, tokenString)

	// Parse the token to verify claims
	token, err := jwt.ParseWithClaims(tokenString, &services.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(cfg.JWTSecret), nil
	})
	require.NoError(t, err)
	require.True(t, token.Valid)

	claims, ok := token.Claims.(*services.JWTClaims)
	require.True(t, ok)

	// Verify token contains correct claims
	assert.Equal(t, testUser.UserID, claims.UserID)
	assert.Equal(t, testUser.Username, claims.Username)
	assert.Equal(t, testUser.Email, claims.Email)
	assert.NotEmpty(t, claims.TokenID)
	assert.Equal(t, "transaction-tracker", claims.Issuer)
	assert.Equal(t, "user:"+testUser.UserID.String(), claims.Subject)

	// Check expiration time is approximately 24 hours from creation
	expectedExpiration := time.Now().Add(24 * time.Hour)
	actualExpiration := claims.ExpiresAt.Time
	timeDiff := actualExpiration.Sub(expectedExpiration).Abs()
	assert.True(t, timeDiff < time.Minute, "Expiration time should be within 1 minute of expected")

	// Verify token is stored in database with composite hash
	var jwtToken models.JWTToken
	err = db.Where("user_id = ?", testUser.UserID).First(&jwtToken).Error
	assert.NoError(t, err)
	assert.Equal(t, testUser.UserID, jwtToken.UserID)
	assert.NotEmpty(t, jwtToken.TokenHash)
	assert.NotEmpty(t, jwtToken.DeviceInfo)
}

func TestJWTService_GenerateToken_InvalidUser(t *testing.T) {
	_, _, jwtService, _ := setupJWTServiceTest(t)
	deviceInfo := createTestDeviceInfo()

	// Test with nil user
	tokenString, err := jwtService.GenerateToken(nil, deviceInfo)
	assert.Error(t, err)
	assert.Empty(t, tokenString)
}

// Test 1.2: Token Validation Tests
func TestJWTService_ValidateToken(t *testing.T) {
	db, _, jwtService, testUser := setupJWTServiceTest(t)
	deviceInfo := createTestDeviceInfo()

	// Generate a valid token
	tokenString, err := jwtService.GenerateToken(testUser, deviceInfo)
	require.NoError(t, err)

	// Test successful validation
	claims, err := jwtService.ValidateToken(tokenString)
	assert.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, testUser.UserID, claims.UserID)
	assert.Equal(t, testUser.Username, claims.Username)
	assert.Equal(t, testUser.Email, claims.Email)

	// Verify last_used_at timestamp is updated
	var jwtToken models.JWTToken
	err = db.Where("user_id = ?", testUser.UserID).First(&jwtToken).Error
	require.NoError(t, err)
	assert.NotNil(t, jwtToken.LastUsedAt)
}

func TestJWTService_ValidateToken_InvalidToken(t *testing.T) {
	_, _, jwtService, _ := setupJWTServiceTest(t)

	// Test with invalid token string
	claims, err := jwtService.ValidateToken("invalid.token.string")
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestJWTService_ValidateToken_ExpiredToken(t *testing.T) {
	_, cfg, jwtService, testUser := setupJWTServiceTest(t)
	deviceInfo := createTestDeviceInfo()

	// Create an expired token by setting short expiration in config
	cfg.JWTExpirationHours = -1 // Expired 1 hour ago

	tokenString, err := jwtService.GenerateToken(testUser, deviceInfo)
	require.NoError(t, err)

	// Reset config for validation
	cfg.JWTExpirationHours = 24

	// Try to validate expired token
	claims, err := jwtService.ValidateToken(tokenString)
	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.Contains(t, err.Error(), "expired")
}

func TestJWTService_ValidateToken_RevokedToken(t *testing.T) {
	_, _, jwtService, testUser := setupJWTServiceTest(t)
	deviceInfo := createTestDeviceInfo()

	// Generate a valid token
	tokenString, err := jwtService.GenerateToken(testUser, deviceInfo)
	require.NoError(t, err)

	// Revoke the token
	err = jwtService.RevokeToken(tokenString)
	require.NoError(t, err)

	// Try to validate revoked token
	claims, err := jwtService.ValidateToken(tokenString)
	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.Contains(t, err.Error(), "revoked")
}

func TestJWTService_ValidateToken_TamperedToken(t *testing.T) {
	_, _, jwtService, testUser := setupJWTServiceTest(t)
	deviceInfo := createTestDeviceInfo()

	// Generate a valid token
	tokenString, err := jwtService.GenerateToken(testUser, deviceInfo)
	require.NoError(t, err)

	// Tamper with the token (change last character)
	tamperedToken := tokenString[:len(tokenString)-1] + "X"

	// Try to validate tampered token
	claims, err := jwtService.ValidateToken(tamperedToken)
	assert.Error(t, err)
	assert.Nil(t, claims)
}

// Test 1.3: Token Revocation Tests
func TestJWTService_RevokeToken(t *testing.T) {
	_, _, jwtService, testUser := setupJWTServiceTest(t)
	deviceInfo := createTestDeviceInfo()

	// Generate a token
	tokenString, err := jwtService.GenerateToken(testUser, deviceInfo)
	require.NoError(t, err)

	// Verify token is valid initially
	claims, err := jwtService.ValidateToken(tokenString)
	assert.NoError(t, err)
	assert.NotNil(t, claims)

	// Revoke the token
	err = jwtService.RevokeToken(tokenString)
	assert.NoError(t, err)

	// Verify token is now invalid
	claims, err = jwtService.ValidateToken(tokenString)
	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.Contains(t, err.Error(), "revoked")
}

// Test 1.4: Token ID Extraction Tests
func TestJWTService_ExtractTokenID(t *testing.T) {
	_, _, jwtService, testUser := setupJWTServiceTest(t)
	deviceInfo := createTestDeviceInfo()

	// Generate a token
	tokenString, err := jwtService.GenerateToken(testUser, deviceInfo)
	require.NoError(t, err)

	// Extract token ID
	tokenID, err := jwtService.ExtractTokenID(tokenString)
	assert.NoError(t, err)
	assert.NotEmpty(t, tokenID)

	// Verify the extracted ID matches the original
	claims, err := jwtService.ValidateToken(tokenString)
	require.NoError(t, err)
	assert.Equal(t, claims.TokenID, tokenID)
}

func TestJWTService_ExtractTokenID_MalformedToken(t *testing.T) {
	_, _, jwtService, _ := setupJWTServiceTest(t)

	// Test with malformed token
	tokenID, err := jwtService.ExtractTokenID("malformed.token")
	assert.Error(t, err)
	assert.Empty(t, tokenID)
	assert.Contains(t, err.Error(), "failed to parse token")
}

// Test helper functions and edge cases
func TestJWTService_GetActiveTokens(t *testing.T) {
	_, _, jwtService, testUser := setupJWTServiceTest(t)
	deviceInfo := createTestDeviceInfo()

	// Initially no active tokens
	tokens, err := jwtService.GetActiveTokens(testUser.UserID)
	assert.NoError(t, err)
	assert.Len(t, tokens, 0)

	// Generate a token
	_, err = jwtService.GenerateToken(testUser, deviceInfo)
	require.NoError(t, err)

	// Should have one active token
	tokens, err = jwtService.GetActiveTokens(testUser.UserID)
	assert.NoError(t, err)
	assert.Len(t, tokens, 1)
	assert.Equal(t, testUser.UserID, tokens[0].UserID)
}

func TestJWTService_CleanupExpiredTokens(t *testing.T) {
	db, cfg, jwtService, testUser := setupJWTServiceTest(t)
	deviceInfo := createTestDeviceInfo()

	// Create an expired token by manipulating the database directly
	cfg.JWTExpirationHours = -1 // Generate expired token
	_, err := jwtService.GenerateToken(testUser, deviceInfo)
	require.NoError(t, err)

	// Reset config
	cfg.JWTExpirationHours = 24

	// Verify token exists in database
	var count int64
	err = db.Model(&models.JWTToken{}).Where("user_id = ?", testUser.UserID).Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// Run cleanup
	err = jwtService.CleanupExpiredTokens()
	assert.NoError(t, err)

	// Verify expired token is removed
	err = db.Model(&models.JWTToken{}).Where("user_id = ?", testUser.UserID).Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}
