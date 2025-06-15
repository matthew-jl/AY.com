package utils

import (
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMain can be used to set up environment variables for tests if needed
func TestMain(m *testing.M) {
	// Set a consistent JWT secret for tests if not already set by CI/environment
	// This ensures jwtSecretKey in jwt.go is initialized predictably.
	if os.Getenv("JWT_SECRET_KEY") == "" {
		os.Setenv("JWT_SECRET_KEY", "test-secret-for-jwt-utils")
	}
	// Re-initialize our jwtSecretKey in the jwt.go package if it uses init()
	// This is a bit of a hack due to package init(). A better way would be to
	// make jwtSecretKey configurable via a function or pass it to GenerateTokens.
	// For this example, we assume init() in jwt.go will pick up the env var.
	// If jwt.go's init has already run, this won't re-init it unless the package structure allows.
	// Let's assume for the simple test that jwtSecretKey in jwt.go gets set.
	// A more robust approach involves making the secret injectable.
	// init() // Re-call init to pick up the env var (if jwt.go's init depends on it)

	code := m.Run()
	os.Exit(code)
}

func TestGenerateTokens(t *testing.T) {
	userID := uint(123)

	accessToken, refreshToken, err := GenerateTokens(userID)

	assert.NoError(t, err, "GenerateTokens should not produce an error")
	assert.NotEmpty(t, accessToken, "Access token should not be empty")
	assert.NotEmpty(t, refreshToken, "Refresh token should not be empty")

	// Optional: Basic parsing to check claims (more involved)
	// Parse Access Token
	accessClaims := jwt.MapClaims{}
	_, err = jwt.ParseWithClaims(accessToken, accessClaims, func(token *jwt.Token) (interface{}, error) {
		return jwtSecretKey, nil // Use the same secret key from the package
	})
	require.NoError(t, err, "Parsing access token should not error")

	// Check some claims in access token
	assert.Equal(t, float64(userID), accessClaims["sub"], "Access token 'sub' claim should match userID")
	assert.Equal(t, "access", accessClaims["typ"], "Access token 'typ' claim should be 'access'")
	expAccess, okAccess := accessClaims["exp"].(float64)
	require.True(t, okAccess, "Access token 'exp' claim should be a number")
	assert.True(t, time.Now().Unix() < int64(expAccess), "Access token should not be expired")

	// Parse Refresh Token
	refreshClaims := jwt.MapClaims{}
	_, err = jwt.ParseWithClaims(refreshToken, refreshClaims, func(token *jwt.Token) (interface{}, error) {
		return jwtSecretKey, nil
	})
	require.NoError(t, err, "Parsing refresh token should not error")
	assert.Equal(t, float64(userID), refreshClaims["sub"], "Refresh token 'sub' claim should match userID")
	assert.Equal(t, "refresh", refreshClaims["typ"], "Refresh token 'typ' claim should be 'refresh'")
}